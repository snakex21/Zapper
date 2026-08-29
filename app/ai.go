package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type aiProfileInput struct {
	PersonID     string      `json:"person_id"`
	Programs     []aiProgram `json:"programs"`
	SameDayPairs [][]string  `json:"same_day_pairs,omitempty"`
	Phases       []aiPhase   `json:"phases"`
}

// Pola liczbowe z oczywistą wartością domyślną są wskaźnikami, żeby odróżnić
// „pole nieobecne” (bierzemy domyślną) od „pole obecne z błędną wartością”
// (odrzucamy). Przy zwykłym unmarshalu obie sytuacje dają w Go zero.
type aiProgram struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Steps                []aiProgramStep `json:"steps"`
	MinGapMinutes        *int            `json:"min_gap_minutes,omitempty"`
	CooldownAfterMinutes *int            `json:"cooldown_after_minutes,omitempty"`
}

type aiProgramStep struct {
	FrequencyHz     float64 `json:"frequency_hz"`
	DurationSeconds uint32  `json:"duration_seconds"`
}

type aiPhase struct {
	Name           string                     `json:"name"`
	Days           int                        `json:"days"`
	EveryDays      *int                       `json:"every_days,omitempty"`
	Program        string                     `json:"program,omitempty"`
	FrequencyHz    *float64                   `json:"frequency_hz,omitempty"`
	DurationSeconds *uint32                   `json:"duration_seconds,omitempty"`
	Steps          []aiProgramStep            `json:"steps,omitempty"`
	Repeat         *int                       `json:"repeat,omitempty"`
	BreakMinutes   *int                       `json:"break_minutes,omitempty"`
	Note           string                     `json:"note,omitempty"`
	Week           map[string]aiSessionChoice `json:"week,omitempty"`
}

type aiSessionChoice struct {
	Program         string           `json:"program"`
	FrequencyHz     *float64         `json:"frequency_hz,omitempty"`
	DurationSeconds *uint32          `json:"duration_seconds,omitempty"`
	Steps           []aiProgramStep  `json:"steps,omitempty"`
	Repeat          *int             `json:"repeat,omitempty"`
	BreakMinutes    *int             `json:"break_minutes,omitempty"`
	Note            string           `json:"note,omitempty"`
}

// optionalInt zwraca wartość pola, jeśli AI je podało, albo wartość domyślną.
func optionalInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func (choice *aiSessionChoice) UnmarshalJSON(data []byte) error {
	var program string
	if err := json.Unmarshal(data, &program); err == nil {
		choice.Program = program
		return nil
	}
	type alias aiSessionChoice
	return json.Unmarshal(data, (*alias)(choice))
}

type aiProgramRuntime struct {
	Name       string
	Steps      []DeviceStep
	Scheduling SessionScheduling
}

func (a *Application) GenerateAIContext(request AIContextRequest) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := request.PersonIDs
	if len(ids) == 0 && strings.TrimSpace(request.PersonID) != "" {
		ids = []string{strings.TrimSpace(request.PersonID)}
	}
	if len(ids) == 0 {
		return "", errors.New("wybierz co najmniej jedną osobę")
	}
	mode := strings.ToLower(strings.TrimSpace(request.Mode))
	if mode != "continuation" {
		mode = "new"
	}
	multi := len(ids) > 1
	var builder strings.Builder
	if multi {
		fmt.Fprintf(&builder, "Poniższy kontekst zawiera %d niezależnych bloków — po jednym dla każdej osoby. Dla każdej osoby przygotuj jeden obiekt JSON z jej person_id, a wszystkie te obiekty wróć jako JEDNĄ tablicę.\n\n", len(ids))
	}
	for index, id := range ids {
		person, exists := a.personByIDLocked(strings.TrimSpace(id))
		if !exists {
			return "", fmt.Errorf("nie znaleziono osoby o ID %s", id)
		}
		if multi && index > 0 {
			builder.WriteString("\n---\n\n")
		}
		block, err := a.generateAIContextBlockLocked(person, mode, multi)
		if err != nil {
			return "", err
		}
		builder.WriteString(block)
	}
	return builder.String(), nil
}

// markdownCell zabezpiecza tekst wstawiany do komórki tabeli markdown.
func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func (a *Application) generateAIContextBlockLocked(person Person, mode string, multi bool) (string, error) {
	var active *Profile
	for index := range a.config.Profiles {
		if profilePersonID(a.config.Profiles[index]) == person.ID {
			copy := a.config.Profiles[index]
			active = &copy
			break
		}
	}

	type historySummary struct {
		Name   string
		Count  int
		Latest string
	}
	history := map[string]*historySummary{}
	addCompletion := func(completion SessionCompletion) {
		key := readableID(completion.ProgramID)
		if key == "" {
			key = readableID(completion.Therapy)
		}
		if key == "" {
			key = "program_bez_id"
		}
		entry := history[key]
		if entry == nil {
			entry = &historySummary{Name: valueOr(completion.Therapy, key)}
			history[key] = entry
		}
		entry.Count++
		date := strings.TrimSpace(completion.CompletedAt)
		if date > entry.Latest {
			entry.Latest = date
		}
	}
	for _, completion := range a.progress.Completions {
		if completion.PersonID == person.ID || completion.ProfileID == person.ID {
			addCompletion(completion)
		}
	}
	for _, archived := range a.archive.Profiles {
		if profilePersonID(archived.Profile) != person.ID || archived.Progress == nil {
			continue
		}
		for _, completion := range archived.Progress.Completions {
			addCompletion(completion)
		}
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "# Osoba: %s\n\n", person.Name)
	fmt.Fprintf(&builder, "- **person_id:** `%s`\n", person.ID)
	if mode == "continuation" {
		builder.WriteString("- **Zadanie:** przygotuj kontynuację obecnego programu.\n\n")
	} else {
		builder.WriteString("- **Zadanie:** przygotuj nowy program.\n\n")
	}
	if multi {
		builder.WriteString("> Ten kontekst jest napisany w Markdown, żeby był czytelny. **Twoja odpowiedź ma być czymś innym: tablicą obiektów JSON — po jednym obiekcie na każdą osobę z tego kontekstu, w tej samej kolejności co bloki.** Bez Markdown, bez bloku kodu, bez komentarzy i bez żadnego tekstu przed ani po nim. Pierwszy znak odpowiedzi to `[`, ostatni to `]`.\n\n")
	} else {
		builder.WriteString("> Ten kontekst jest napisany w Markdown, żeby był czytelny. **Twoja odpowiedź ma być czymś innym: samym obiektem JSON** — bez Markdown, bez bloku kodu, bez komentarzy i bez żadnego tekstu przed ani po nim. Pierwszy znak odpowiedzi to `{`, ostatni to `}`.\n\n")
	}

	keys := make([]string, 0, len(history))
	for key := range history {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	builder.WriteString("## Skrót wykonanych programów\n\n")
	if len(keys) == 0 {
		builder.WriteString("Brak zapisanych wykonanych sesji.\n\n")
	} else {
		builder.WriteString("| Program | id | Liczba sesji | Ostatnia data |\n| --- | --- | ---: | --- |\n")
		for _, key := range keys {
			entry := history[key]
			latest := "brak daty"
			if len(entry.Latest) >= 10 {
				latest = entry.Latest[:10]
			}
			fmt.Fprintf(&builder, "| %s | `%s` | %d | %s |\n", markdownCell(entry.Name), markdownCell(key), entry.Count, markdownCell(latest))
		}
		builder.WriteString("\n")
	}

	if mode == "continuation" {
		builder.WriteString("## Obecny program\n\n")
		if active == nil {
			builder.WriteString("Brak aktywnego programu; przygotuj nowy.\n\n")
		} else {
			fmt.Fprintf(&builder, "- **run_id:** `%s`\n- **Data startu:** %s\n\n", active.RunID, active.StartDate)
			builder.WriteString("### Fazy\n\n")
			if len(active.Phases) == 0 {
				builder.WriteString("- Brak faz.\n\n")
			} else {
				for _, phase := range active.Phases {
					fmt.Fprintf(&builder, "- **%s** — %d dni, tryb `%s`\n", phase.Name, phase.DurationDays, phase.Mode)
				}
				builder.WriteString("\n")
			}
			plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
			rows := make([]string, 0, len(plans))
			for _, plan := range plans {
				if plan.PersonID != person.ID || plan.Status != "session" {
					continue
				}
				status := "oczekuje"
				if plan.Done {
					status = "wykonana"
				} else if plan.Paused {
					status = fmt.Sprintf("wstrzymana, pozostało %d s", plan.RemainingSeconds)
				} else if plan.Overdue {
					status = "zaległa"
				}
				rows = append(rows, fmt.Sprintf("- %s — **%s** — %s — *%s*\n", plan.PlannedDate, plan.PhaseName, plan.Program, status))
			}
			builder.WriteString("### Zaplanowane sesje\n\n")
			if len(rows) == 0 {
				builder.WriteString("- Brak zaplanowanych sesji.\n\n")
			} else {
				for _, row := range rows {
					builder.WriteString(row)
				}
				builder.WriteString("\n")
			}
		}
	}

	formatIntro := "Odeślij dokładnie taki obiekt (sam JSON, bez otaczającego bloku kodu). Pola opcjonalne możesz pominąć:"
	if multi {
		formatIntro = "Odeślij dokładnie taką TABLICĘ (sam JSON, bez otaczającego bloku kodu) — jeden element na osobę, w tej samej kolejności co bloki kontekstu. Każdy element to obiekt jak poniżej, z właściwym dla tej osoby person_id:"
	}
	example := `{
  "person_id": "` + person.ID + `",
  "phases": [
    {
      "name": "Start",
      "days": 7,
      "frequency_hz": 30000,
      "duration_seconds": 420
    },
    {
      "name": "Plan tygodniowy",
      "days": 28,
      "week": {
        "Monday": {"frequency_hz": 30000, "duration_seconds": 420},
        "Friday": {"frequency_hz": 727, "duration_seconds": 180, "repeat": 2, "break_minutes": 20}
      }
    }
  ]
}`
	builder.WriteString("## Format odpowiedzi\n\n")
	builder.WriteString(formatIntro + "\n\n")
	if multi {
		builder.WriteString("```json\n[\n  " + example + "\n]\n```\n\n")
	} else {
		builder.WriteString("```json\n" + example + "\n```\n\n")
	}
	builder.WriteString("## Zasady formatu\n\n- Wymagane są tylko: `person_id` oraz `phases` (z `name`, `days` i sesją). Resztę pomiń, jeśli nie potrzebujesz.\n")
	builder.WriteString("- Sesję opisujesz parą `frequency_hz` + `duration_seconds` — bezpośrednio w fazie (tryb co kilka dni) albo w dniu tygodnia w `week`. To minimalna forma, używaj jej, gdy nie potrzebujesz odstępów między powtórzeniami.\n")
	builder.WriteString("- Kilka częstotliwości w jednej sesji: opcjonalna tablica `steps` z tymi samymi polami (max 12 kroków).\n")
	builder.WriteString("- Pominięte pola dostają wartości domyślne: `repeat` = 1, `break_minutes` = 0, `every_days` = 1 (codziennie), `note` = pusta. Pominięty `cooldown_after_minutes` oznacza 0, czyli brak odstępu przed innym programem — nie pomijaj go przypadkiem.\n")
	builder.WriteString("- `frequency_hz` jest w hercach, np. 7.83, 30000 lub 1000000 — aplikacja sama przelicza to na miliherce.\n")
	builder.WriteString("- Zakresy dla wartości, które podasz: `duration_seconds` 1–86400, `every_days` 1–366, `repeat` 1–12, `break_minutes` 0–1440.\n")
	builder.WriteString("- Faza ma albo `every_days` + sesję w sobie, albo `week` z angielskimi kluczami Monday–Sunday. W `week` wartością jest obiekt sesji.\n")
	builder.WriteString("- Gdy potrzebujesz odstępów `min_gap_minutes`/`cooldown_after_minutes` albo sesji łączonych tego samego dnia: zdefiniuj nazwane programy w `programs` (każdy z `id`, `name`, `steps` i opcjonalnymi odstępami) i w fazie/dniu podaj tylko `\"program\": \"id\"`. `same_day_pairs` podaj raz na parę — aplikacja domyka relację w obie strony.\n")
	builder.WriteString("- Każdy program użyty w fazach musi znajdować się w tablicy `programs`.\n")
	builder.WriteString("- Trzymaj się dokładnie pól z tego opisu i nie wymyślaj własnych. Pole spoza schematu nie przerwie importu, ale zostanie zignorowane i pokazane użytkownikowi jako ostrzeżenie — pilnuj też literówek w nazwach pól.\n\n")
	builder.WriteString("## Zasady odpowiedzi\n\n")
	if multi {
		builder.WriteString("- Odpowiedz **wyłącznie** tablicą JSON — po jednym obiekcie na każdą osobę, w tej samej kolejności co bloki kontekstu. Bez Markdown, bez bloku kodu, bez nagłówków, bez komentarzy, bez wyjaśnień przed ani po.\n")
		builder.WriteString("- Odpowiedź musi zaczynać się znakiem `[` i kończyć znakiem `]`.\n")
	} else {
		builder.WriteString("- Odpowiedz **wyłącznie** jednym obiektem JSON. Bez Markdown, bez bloku kodu, bez nagłówków, bez komentarzy, bez wyjaśnień przed ani po.\n")
		builder.WriteString("- Odpowiedź musi zaczynać się znakiem `{` i kończyć znakiem `}`.\n")
	}
	builder.WriteString("- Nie zmieniaj wartości `person_id`.\n")
	return builder.String(), nil
}

func (a *Application) PreviewAIProfile(raw string) (AIImportPreviewBatch, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	imports, err := a.parseAIPayloadLocked(raw)
	if err != nil {
		return AIImportPreviewBatch{}, err
	}
	// Dla pojedynczego obiektu ścieżki nieznanych pól zostają dokładnie takie
	// jak dawniej (np. phases[0].repeats). Dla tablicy osób `[0].phases[...]`
	// rozdzielamy je na podgląd właściwej osoby.
	unknownByPerson := map[int][]string{}
	if len(imports) > 1 {
		for _, path := range collectUnknownFields(raw, []aiProfileInput{}) {
			index := -1
			if strings.HasPrefix(path, "[") {
				if close := strings.Index(path, "]"); close > 0 {
					if parsed, convErr := strconv.Atoi(path[1:close]); convErr == nil && parsed >= 0 && parsed < len(imports) {
						index = parsed
					}
				}
			}
			if index < 0 {
				continue
			}
			trimmed := strings.TrimPrefix(path[strings.Index(path, "]")+1:], ".")
			unknownByPerson[index] = append(unknownByPerson[index], trimmed)
		}
	}
	batch := AIImportPreviewBatch{Persons: make([]AIImportPreview, 0, len(imports))}
	for index, item := range imports {
		person, _ := a.personByIDLocked(item.Input.PersonID)
		preview := AIImportPreview{
			PersonID:   person.ID,
			PersonName: person.Name,
			PhaseCount: len(item.Profile.Phases),
			Summary:    make([]string, 0, len(item.Profile.Phases)),
		}
		preview.ProgramCount = aiProgramCount(item.Input)
		preview.Warnings = aiSchedulingWarnings(item.Input)
		if len(imports) == 1 {
			preview.UnknownFields = collectUnknownFields(raw, aiProfileInput{})
		} else {
			preview.UnknownFields = unknownByPerson[index]
		}
		for _, current := range a.config.Profiles {
			if profilePersonID(current) == person.ID {
				preview.ReplacesActive = true
				break
			}
		}
		for _, phase := range item.Profile.Phases {
			preview.TotalDays += phase.DurationDays
			preview.Summary = append(preview.Summary, fmt.Sprintf("%s — %d dni — %s", phase.Name, phase.DurationDays, phase.Mode))
		}
		batch.Persons = append(batch.Persons, preview)
	}
	return batch, nil
}

func aiSchedulingWarnings(input aiProfileInput) []string {
	warnings := make([]string, 0)
	seen := map[string]bool{}
	add := func(message string) {
		if !seen[message] {
			seen[message] = true
			warnings = append(warnings, message)
		}
	}
	for _, program := range input.Programs {
		if program.CooldownAfterMinutes == nil {
			add(fmt.Sprintf("Program %q nie ma cooldown_after_minutes — po nim inny program może być dostępny od razu.", valueOr(program.Name, program.ID)))
		}
	}
	inline := false
	for _, phase := range input.Phases {
		if len(phase.Week) > 0 {
			for _, choice := range phase.Week {
				if strings.TrimSpace(choice.Program) == "" {
					inline = true
				}
			}
			continue
		}
		if strings.TrimSpace(phase.Program) == "" {
			inline = true
		}
	}
	if inline {
		add("Co najmniej jedna sesja jest wpisana bezpośrednio częstotliwością, więc nie ma cooldownu między różnymi programami. Użyj nazwanego programu, jeśli odstęp ma być pilnowany.")
	}
	return warnings
}

// aiProgramCount liczy różne sesje użyte w fazach (nazwane programy albo
// pary częstotliwość + czas) — to liczba, którą podgląd pokazuje jako
// "programy".
func aiProgramCount(input aiProfileInput) int {
	labels := map[string]bool{}
	count := 0
	mark := func(label string) {
		if label != "" && !labels[label] {
			labels[label] = true
			count++
		}
	}
	for _, phase := range input.Phases {
		if len(phase.Week) > 0 {
			for _, choice := range phase.Week {
				if choice.Program != "" {
					mark(readableID(choice.Program))
					continue
				}
				steps, err := stepsFromAISession(choice.FrequencyHz, choice.DurationSeconds, choice.Steps)
				if err != nil {
					continue
				}
				mark(formatFrequencyLabel(steps))
			}
			continue
		}
		if phase.Program != "" {
			mark(readableID(phase.Program))
			continue
		}
		steps, err := stepsFromAISession(phase.FrequencyHz, phase.DurationSeconds, phase.Steps)
		if err != nil {
			continue
		}
		mark(formatFrequencyLabel(steps))
	}
	return count
}

func (a *Application) ApplyAIProfile(raw string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	imports, err := a.parseAIPayloadLocked(raw)
	if err != nil {
		return Snapshot{}, err
	}
	// Najpierw walidujemy CAŁY docelowy stan — dopiero gdy wszystko jest
	// poprawne, cokolwiek ruszamy na dysku. Inaczej jeden błąd w ostatniej
	// osobie zostawiłby połówkę już zarchiwizowanych programów.
	replacedByPerson := map[string]bool{}
	for _, item := range imports {
		replacedByPerson[item.Input.PersonID] = true
	}
	targetProfiles := make([]Profile, 0, len(a.config.Profiles)+len(imports))
	for _, current := range a.config.Profiles {
		if replacedByPerson[profilePersonID(current)] {
			continue
		}
		targetProfiles = append(targetProfiles, current)
	}
	for _, item := range imports {
		targetProfiles = append(targetProfiles, item.Profile)
	}
	if _, err := validateConfig(&Config{StartDate: a.config.StartDate, Profiles: targetProfiles}, validationModeSave); err != nil {
		return Snapshot{}, err
	}
	importedProfiles := make([]Profile, 0, len(imports))
	for _, item := range imports {
		person, _ := a.personByIDLocked(item.Input.PersonID)
		existingIndex := -1
		for index, current := range a.config.Profiles {
			if profilePersonID(current) == person.ID {
				existingIndex = index
				break
			}
		}
		if existingIndex >= 0 {
			if err := a.archiveProfileAtLocked(existingIndex, "Zastąpiony nowym profilem zaimportowanym z AI"); err != nil {
				return Snapshot{}, err
			}
		}
		importedProfiles = append(importedProfiles, item.Profile)
		for index := range a.persons.Persons {
			if a.persons.Persons[index].ID == person.ID {
				a.persons.Persons[index].Active = true
			}
		}
	}
	a.config.Profiles = append(a.config.Profiles, importedProfiles...)
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

// stripJSONFence usuwa opakowanie ```…``` , którym modele lubią otaczać JSON.
func stripJSONFence(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 3 {
			lines = lines[1 : len(lines)-1]
			raw = strings.Join(lines, "\n")
		}
	}
	return raw
}

// aiPersonImport łączy wejście od AI (osoba + programy + fazy) z gotowym
// profilem, żeby zapis importu wieloosobowego widział całość naraz.
type aiPersonImport struct {
	Input   aiProfileInput
	Profile Profile
}

// parseAIPayloadLocked parsuje odpowiedź AI podaną jako pojedynczy obiekt
// (jedna osoba) ALBO jako tablicę obiektów (osoba na element) i dla każdego
// elementu buduje kompletny profil. Dzięki temu jeden import może od razu
// założyć programy wielu osobom z jednego pliku tekstowego.
func (a *Application) parseAIPayloadLocked(raw string) ([]aiPersonImport, error) {
	raw = stripJSONFence(raw)
	if strings.HasPrefix(strings.TrimSpace(raw), "[") {
		var inputs []aiProfileInput
		decoder := json.NewDecoder(bytes.NewBufferString(raw))
		// Nieznane pola są tolerowane (nie wywalają importu), ale zbieramy je
		// osobno w collectUnknownFields i pokazujemy użytkownikowi w podglądzie,
		// żeby literówka w nazwie pola nie przeszła niezauważona.
		if err := decoder.Decode(&inputs); err != nil {
			return nil, fmt.Errorf("nieprawidłowy JSON od AI: %w", err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			if err == nil {
				return nil, errors.New("JSON zawiera więcej niż jedną tablicę")
			}
			return nil, fmt.Errorf("dodatkowe dane za tablicą JSON: %w", err)
		}
		if len(inputs) == 0 {
			return nil, errors.New("tablica odpowiedzi AI jest pusta")
		}
		imports := make([]aiPersonImport, 0, len(inputs))
		seen := map[string]bool{}
		for index, input := range inputs {
			parsedInput, profile, err := a.buildAIProfileFromInputLocked(input, index)
			if err != nil {
				return nil, fmt.Errorf("osoba %q: %w", parsedInput.PersonID, err)
			}
			if seen[parsedInput.PersonID] {
				return nil, fmt.Errorf("osoba %q występuje więcej niż raz w odpowiedzi AI", parsedInput.PersonID)
			}
			seen[parsedInput.PersonID] = true
			imports = append(imports, aiPersonImport{Input: parsedInput, Profile: profile})
		}
		return imports, nil
	}
	var input aiProfileInput
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	// Nieznane pola są tolerowane (nie wywalają importu), ale zbieramy je
	// osobno w collectUnknownFields i pokazujemy użytkownikowi w podglądzie,
	// żeby literówka w nazwie pola nie przeszła niezauważona.
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("nieprawidłowy JSON od AI: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("JSON zawiera więcej niż jeden obiekt")
		}
		return nil, fmt.Errorf("dodatkowe dane za obiektem JSON: %w", err)
	}
	parsedInput, profile, err := a.buildAIProfileFromInputLocked(input, -1)
	if err != nil {
		return nil, err
	}
	return []aiPersonImport{{Input: parsedInput, Profile: profile}}, nil
}

// parseAIProfileLocked parsuje odpowiedź dla pojedynczej osoby — wsteczna
// zgodność dla kodu, który trzymał się starej sygnatury.
func (a *Application) parseAIProfileLocked(raw string) (aiProfileInput, Profile, error) {
	imports, err := a.parseAIPayloadLocked(raw)
	if err != nil {
		return aiProfileInput{}, Profile{}, err
	}
	if len(imports) != 1 {
		return aiProfileInput{}, Profile{}, errors.New("oczekiwano dokładnie jednego obiektu JSON")
	}
	return imports[0].Input, imports[0].Profile, nil
}

// buildAIProfileFromInputLocked zamienia wejście AI (programy + fazy) w
// kompletny profil jednej osoby. element to indeks w tablicy wieloosobowej
// (−1, gdy odpowiedź zawierała pojedynczy obiekt).
func (a *Application) buildAIProfileFromInputLocked(input aiProfileInput, element int) (aiProfileInput, Profile, error) {
	input.PersonID = readableID(input.PersonID)
	person, exists := a.personByIDLocked(input.PersonID)
	if !exists {
		if element >= 0 {
			return input, Profile{}, fmt.Errorf("persons.json nie zawiera osoby %q (element %d)", input.PersonID, element+1)
		}
		return input, Profile{}, fmt.Errorf("persons.json nie zawiera osoby %q", input.PersonID)
	}
	if len(input.Phases) == 0 {
		return input, Profile{}, errors.New("tablica phases jest pusta")
	}
	// Tablica programs jest opcjonalna: sesje można opisać wprost parami
	// frequency_hz + duration_seconds w fazie lub dniu tygodnia. programs
	// zostaje dla nazwanych programów z własnymi odstępami (min_gap/cooldown).

	// Relację wystarczy podać z jednej strony — domykamy ją symetrycznie tutaj.
	// Zapis obustronny i powtórzone pary trafiają do tego samego zbioru, więc
	// domknięcie jest idempotentne i niczego nie duplikuje.
	compatible := map[string]map[string]bool{}
	for index, pair := range input.SameDayPairs {
		if len(pair) != 2 {
			return input, Profile{}, fmt.Errorf("same_day_pairs[%d] musi zawierać dokładnie dwa ID", index)
		}
		first, second := readableID(pair[0]), readableID(pair[1])
		if first == "" || second == "" || first == second {
			return input, Profile{}, fmt.Errorf("same_day_pairs[%d] zawiera nieprawidłową parę", index)
		}
		if compatible[first] == nil {
			compatible[first] = map[string]bool{}
		}
		if compatible[second] == nil {
			compatible[second] = map[string]bool{}
		}
		compatible[first][second] = true
		compatible[second][first] = true
	}

	programs := map[string]aiProgramRuntime{}
	for index, program := range input.Programs {
		program.ID = readableID(program.ID)
		program.Name = strings.TrimSpace(program.Name)
		if program.ID == "" || program.Name == "" {
			return input, Profile{}, fmt.Errorf("program %d musi mieć id i name", index+1)
		}
		if _, duplicate := programs[program.ID]; duplicate {
			return input, Profile{}, fmt.Errorf("program %q występuje więcej niż raz", program.ID)
		}
		if len(program.Steps) < 1 || len(program.Steps) > 12 {
			return input, Profile{}, fmt.Errorf("program %q musi mieć 1–12 kroków", program.ID)
		}
		steps := make([]DeviceStep, 0, len(program.Steps))
		for stepIndex, step := range program.Steps {
			if math.IsNaN(step.FrequencyHz) || math.IsInf(step.FrequencyHz, 0) || step.FrequencyHz < 0.1 || step.FrequencyHz > 4_000_000 {
				return input, Profile{}, fmt.Errorf("program %q, krok %d: frequency_hz musi mieścić się w zakresie 0.1–4000000", program.ID, stepIndex+1)
			}
			deviceStep := DeviceStep{FrequencyMilliHz: uint64(math.Round(step.FrequencyHz * 1000)), DurationSeconds: step.DurationSeconds}
			if err := validateDeviceSteps([]DeviceStep{deviceStep}); err != nil {
				return input, Profile{}, fmt.Errorf("program %q, krok %d: %w", program.ID, stepIndex+1, err)
			}
			steps = append(steps, deviceStep)
		}
		minGap := optionalInt(program.MinGapMinutes, 0)
		cooldown := optionalInt(program.CooldownAfterMinutes, 0)
		if minGap < 0 || minGap > 43200 || cooldown < 0 || cooldown > 43200 {
			return input, Profile{}, fmt.Errorf("program %q ma odstęp poza zakresem 0–43200 min", program.ID)
		}
		sameDay := make([]string, 0, len(compatible[program.ID]))
		for value := range compatible[program.ID] {
			sameDay = append(sameDay, value)
		}
		sort.Strings(sameDay)
		programs[program.ID] = aiProgramRuntime{
			Name:       program.Name,
			Steps:      steps,
			Scheduling: SessionScheduling{ProgramID: program.ID, Repetitions: 1, MinGapMinutes: minGap, CooldownAfterMinutes: cooldown, SameDayWith: sameDay},
		}
	}
	for first, values := range compatible {
		if _, exists := programs[first]; !exists {
			return input, Profile{}, fmt.Errorf("same_day_pairs odwołuje się do nieznanego programu %q", first)
		}
		for second := range values {
			if _, exists := programs[second]; !exists {
				return input, Profile{}, fmt.Errorf("same_day_pairs odwołuje się do nieznanego programu %q", second)
			}
		}
	}

	profile := Profile{ID: person.ID, PersonID: person.ID, RunID: newRunID(), StartDate: a.now().Format("2006-01-02"), Name: person.Name, Phases: make([]Phase, 0, len(input.Phases))}
	for index, source := range input.Phases {
		source.Name = strings.TrimSpace(source.Name)
		if source.Name == "" || source.Days < 1 || source.Days > 3650 {
			return input, Profile{}, fmt.Errorf("faza %d musi mieć nazwę oraz days w zakresie 1–3650", index+1)
		}
		phase := Phase{ID: stableID("phase", person.ID+"|"+source.Name, index), Name: source.Name, DurationDays: source.Days}
		if len(source.Week) > 0 {
			phase.Mode = "weekly"
			phase.Schedule = map[string]DailyPlan{}
			for day, choice := range source.Week {
				weekday, valid := normalizeAIWeekday(day)
				if !valid {
					return input, Profile{}, fmt.Errorf("faza %q zawiera nieznany dzień %q", source.Name, day)
				}
				daily, err := aiDailyPlan(choice, programs)
				if err != nil {
					return input, Profile{}, fmt.Errorf("faza %q, %s: %w", source.Name, weekday, err)
				}
				phase.Schedule[weekday] = daily
			}
		} else {
			phase.Mode = "interval"
			// Brak every_days oznacza sesję codziennie; podana wartość musi być w zakresie.
			everyDays := optionalInt(source.EveryDays, 1)
			if everyDays < 1 || everyDays > 366 {
				return input, Profile{}, fmt.Errorf("faza %q: every_days musi mieścić się w zakresie 1–366", source.Name)
			}
			choice := aiSessionChoice{Program: source.Program, FrequencyHz: source.FrequencyHz, DurationSeconds: source.DurationSeconds, Steps: source.Steps, Repeat: source.Repeat, BreakMinutes: source.BreakMinutes, Note: source.Note}
			daily, err := aiDailyPlan(choice, programs)
			if err != nil {
				return input, Profile{}, fmt.Errorf("faza %q: %w", source.Name, err)
			}
			phase.IntervalGap = everyDays - 1
			phase.Program = daily.Frequency
			phase.Time = daily.Time
			phase.Note = daily.Note
			phase.DeviceSteps = daily.DeviceSteps
			phase.Scheduling = daily.Scheduling
		}
		profile.Phases = append(profile.Phases, phase)
	}
	validationConfig := Config{StartDate: profile.StartDate, Profiles: []Profile{profile}}
	if _, err := validateConfig(&validationConfig, validationModeSave); err != nil {
		return input, Profile{}, err
	}
	profile = validationConfig.Profiles[0]
	return input, profile, nil
}

func aiDailyPlan(choice aiSessionChoice, programs map[string]aiProgramRuntime) (DailyPlan, error) {
	// Pominięte repeat/break_minutes dostają wartości domyślne; podane muszą
	// mieścić się w zakresie — jawne 0 nie jest traktowane jak brak pola.
	repeat := optionalInt(choice.Repeat, 1)
	if repeat < 1 || repeat > 12 {
		return DailyPlan{}, errors.New("repeat musi mieścić się w zakresie 1–12")
	}
	breakMinutes := optionalInt(choice.BreakMinutes, 0)
	if breakMinutes < 0 || breakMinutes > 1440 {
		return DailyPlan{}, errors.New("break_minutes musi mieścić się w zakresie 0–1440")
	}
	if programID := readableID(choice.Program); programID != "" {
		program, exists := programs[programID]
		if !exists {
			return DailyPlan{}, fmt.Errorf("nieznany program %q", choice.Program)
		}
		scheduling := program.Scheduling
		scheduling.Repetitions = repeat
		scheduling.BreakBetweenMinutes = breakMinutes
		var seconds uint32
		for _, step := range program.Steps {
			seconds += step.DurationSeconds
		}
		return DailyPlan{
			Frequency:   program.Name,
			Time:        formatSecondsText(seconds),
			Note:        strings.TrimSpace(choice.Note),
			DeviceSteps: append([]DeviceStep(nil), program.Steps...),
			Scheduling:  scheduling,
		}, nil
	}
	// Tryb bez nazwanych programów: sesja opisana wprost częstotliwością.
	steps, err := stepsFromAISession(choice.FrequencyHz, choice.DurationSeconds, choice.Steps)
	if err != nil {
		return DailyPlan{}, err
	}
	var seconds uint32
	for _, step := range steps {
		seconds += step.DurationSeconds
	}
	return DailyPlan{
		Frequency:   formatFrequencyLabel(steps),
		Time:        formatSecondsText(seconds),
		Note:        strings.TrimSpace(choice.Note),
		DeviceSteps: steps,
		Scheduling: SessionScheduling{
			Repetitions:       repeat,
			BreakBetweenMinutes: breakMinutes,
		},
	}, nil
}

// stepsFromAISession zamienia opis sesji (pojedyncza para częstotliwość + czas
// albo tablica steps) na kroki urządzenia. Bez żadnej z tych form zwraca błąd —
// dzień musi być opisywalny jako sesja.
func stepsFromAISession(frequencyHz *float64, durationSeconds *uint32, steps []aiProgramStep) ([]DeviceStep, error) {
	buildStep := func(step aiProgramStep) (DeviceStep, error) {
		if math.IsNaN(step.FrequencyHz) || math.IsInf(step.FrequencyHz, 0) || step.FrequencyHz < 0.1 || step.FrequencyHz > 4_000_000 {
			return DeviceStep{}, errors.New("frequency_hz musi mieścić się w zakresie 0.1–4000000")
		}
		deviceStep := DeviceStep{FrequencyMilliHz: uint64(math.Round(step.FrequencyHz * 1000)), DurationSeconds: step.DurationSeconds}
		if err := validateDeviceSteps([]DeviceStep{deviceStep}); err != nil {
			return DeviceStep{}, err
		}
		return deviceStep, nil
	}
	if len(steps) > 0 {
		if len(steps) > 12 {
			return nil, errors.New("maksymalnie 12 kroków na sesję")
		}
		deviceSteps := make([]DeviceStep, 0, len(steps))
		for index, step := range steps {
			deviceStep, err := buildStep(step)
			if err != nil {
				return nil, fmt.Errorf("krok %d: %w", index+1, err)
			}
			deviceSteps = append(deviceSteps, deviceStep)
		}
		return deviceSteps, nil
	}
	if frequencyHz == nil {
		return nil, errors.New("dzień musi mieć program albo frequency_hz")
	}
	if durationSeconds == nil {
		return nil, errors.New("przy frequency_hz podaj też duration_seconds")
	}
	deviceStep, err := buildStep(aiProgramStep{FrequencyHz: *frequencyHz, DurationSeconds: *durationSeconds})
	if err != nil {
		return nil, err
	}
	return []DeviceStep{deviceStep}, nil
}

// formatFrequencyLabel tworzy czytelną nazwę sesji z kroków, np. "30 kHz"
// albo "20 Hz + 727 kHz". Używana, gdy sesja nie ma nazwanego programu.
func formatFrequencyLabel(steps []DeviceStep) string {
	labels := make([]string, 0, len(steps))
	for _, step := range steps {
		hz := float64(step.FrequencyMilliHz) / 1000
		if hz >= 1000 && math.Mod(hz, 1000) == 0 {
			labels = append(labels, fmt.Sprintf("%.0f kHz", hz/1000))
		} else if hz >= 1000 {
			labels = append(labels, fmt.Sprintf("%g kHz", hz/1000))
		} else if math.Mod(hz, 1) == 0 {
			labels = append(labels, fmt.Sprintf("%.0f Hz", hz))
		} else {
			labels = append(labels, fmt.Sprintf("%g Hz", hz))
		}
	}
	return strings.Join(labels, " + ")
}

func normalizeAIWeekday(value string) (string, bool) {
	for _, key := range weekdayKeys {
		if strings.EqualFold(strings.TrimSpace(value), key) {
			return key, true
		}
	}
	return "", false
}

func formatSecondsText(seconds uint32) string {
	if seconds%3600 == 0 && seconds >= 3600 {
		return fmt.Sprintf("%d h", seconds/3600)
	}
	if seconds%60 == 0 && seconds >= 60 {
		return fmt.Sprintf("%d min", seconds/60)
	}
	return fmt.Sprintf("%d s", seconds)
}

func (a *Application) archiveProfileAtLocked(index int, reason string) error {
	if index < 0 || index >= len(a.config.Profiles) {
		return errors.New("nie znaleziono aktywnego profilu")
	}
	profile := a.config.Profiles[index]
	_, states := buildOperationalPlans(a.config, a.progress, a.now())
	state := ProfileRuntimeStatus{ProfileID: profile.ID, ProfileName: profile.Name}
	for _, candidate := range states {
		if candidate.ProfileID == profile.ID {
			state = candidate
			break
		}
	}
	archived := ArchivedProfile{
		ArchiveID:      fmt.Sprintf("archive_%s_%d", profile.ID, a.now().UnixNano()),
		FinishedAt:     a.now().Format(time.RFC3339),
		Reason:         reason,
		Profile:        profile,
		OverdueCount:   state.OverdueCount,
		ExtensionDays:  state.ExtensionDays,
		CompletedCount: a.completedCountForProfile(profile),
	}
	profileProgress := progressForProfile(a.progress, profile)
	if err := saveArchiveFolder(a.archivePath, archived, profileProgress); err != nil {
		return err
	}
	archived.Progress = &profileProgress
	if err := removeActiveProgressFile(a.progressPath, profile); err != nil {
		return err
	}
	a.archive.Profiles = append([]ArchivedProfile{archived}, a.archive.Profiles...)
	a.config.Profiles = append(a.config.Profiles[:index], a.config.Profiles[index+1:]...)
	a.progress = progressWithoutProfile(a.progress, profile)
	return nil
}
