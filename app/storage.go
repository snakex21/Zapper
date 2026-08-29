package main

import (
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

func prepareDataDirectory(appDirectory, dataDirectory string) error {
	if err := os.MkdirAll(dataDirectory, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"profiles.json", "persons.json", "progress.json", "device.json"} {
		if err := migrateDataFile(appDirectory, dataDirectory, name); err != nil {
			return err
		}
	}
	return nil
}

func migrateDataFile(appDirectory, dataDirectory, name string) error {
	destination := filepath.Join(dataDirectory, name)
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	legacyPath := filepath.Join(appDirectory, name)
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.Rename(legacyPath, destination); err != nil {
		return fmt.Errorf("nie można przenieść %s do data: %w", name, err)
	}
	return nil
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("nie można odczytać %s: %w", filepath.Base(path), err)
	}
	return nil
}

func saveJSONAtomic(path string, value any) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		backupDir := filepath.Join(directory, "backups")
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return err
		}
		if err := copyFile(path, filepath.Join(backupDir, filepath.Base(path)+".bak")); err != nil {
			return err
		}
	}

	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "    ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("nie można zastąpić %s: %w", filepath.Base(path), err)
	}
	removeTemporary = false
	return nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func loadArchiveFolders(directory string) (ArchiveStore, error) {
	store := ArchiveStore{Profiles: []ArchivedProfile{}}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return store, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return store, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var archived ArchivedProfile
		path := filepath.Join(directory, entry.Name(), "profile.json")
		if err := loadJSON(path, &archived); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return store, err
		}
		var progress Progress
		progressPath := filepath.Join(directory, entry.Name(), "progress.json")
		if err := loadJSON(progressPath, &progress); err == nil {
			archived.Progress = &progress
		} else if !os.IsNotExist(err) {
			return store, err
		}
		store.Profiles = append(store.Profiles, archived)
	}
	sort.SliceStable(store.Profiles, func(i, j int) bool {
		return store.Profiles[i].FinishedAt > store.Profiles[j].FinishedAt
	})
	return store, nil
}

func saveArchiveFolder(directory string, archived ArchivedProfile, progress Progress) error {
	folder := filepath.Join(directory, archived.ArchiveID)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	metadata := archived
	metadata.Progress = nil
	if err := saveJSONAtomic(filepath.Join(folder, "profile.json"), metadata); err != nil {
		return err
	}
	return saveJSONAtomic(filepath.Join(folder, "progress.json"), progress)
}

func activeProgressFile(directory string, profile Profile) string {
	return filepath.Join(directory, profile.ID+"__"+profile.RunID+".json")
}

func loadActiveProgress(directory string, config Config, now time.Time) (Progress, error) {
	result := defaultProgress(now)
	if strings.TrimSpace(config.StartDate) != "" {
		result.StartDate = config.StartDate
		result.TrackingSince = config.StartDate
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return result, err
	}
	loadedAny := false
	for _, profile := range config.Profiles {
		var stored Progress
		if err := loadJSON(activeProgressFile(directory, profile), &stored); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return result, err
		}
		if !loadedAny {
			if stored.StartDate != "" {
				result.StartDate = stored.StartDate
			}
			if stored.TrackingSince != "" {
				result.TrackingSince = stored.TrackingSince
			}
			loadedAny = true
		}
		for date, entries := range stored.History {
			if result.History[date] == nil {
				result.History[date] = map[string]HistoryEntry{}
			}
			for name, entry := range entries {
				result.History[date][name] = entry
			}
		}
		for sessionID, completion := range stored.Completions {
			result.Completions[sessionID] = completion
		}
		for sessionID, dismissed := range stored.DismissedSessions {
			result.DismissedSessions[sessionID] = dismissed
		}
		for sessionID, paused := range stored.PausedSessions {
			result.PausedSessions[sessionID] = paused
		}
	}
	return result, nil
}

func loadManualRuns(path string) (ManualRunStore, error) {
	store := ManualRunStore{Runs: []ManualRun{}}
	if err := loadJSON(path, &store); err != nil {
		if os.IsNotExist(err) {
			return ManualRunStore{Runs: []ManualRun{}}, nil
		}
		return store, err
	}
	if store.Runs == nil {
		store.Runs = []ManualRun{}
	}
	return store, nil
}

func saveActiveProgress(directory string, config Config, progress Progress) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	for _, profile := range config.Profiles {
		if err := saveJSONAtomic(activeProgressFile(directory, profile), progressForProfile(progress, profile)); err != nil {
			return err
		}
	}
	return nil
}

func removeActiveProgressFile(directory string, profile Profile) error {
	err := os.Remove(activeProgressFile(directory, profile))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func preserveMigratedFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	backupDirectory := filepath.Join(filepath.Dir(path), "backups")
	if err := os.MkdirAll(backupDirectory, 0o755); err != nil {
		return err
	}
	destination := filepath.Join(backupDirectory, filepath.Base(path)+".migrated.bak")
	if _, err := os.Stat(destination); err == nil {
		destination = filepath.Join(backupDirectory, fmt.Sprintf("%s.%d.migrated.bak", filepath.Base(path), time.Now().UnixNano()))
	}
	return os.Rename(path, destination)
}

func defaultConfig(now time.Time) Config {
	return Config{
		StartDate: now.Format("2006-01-02"),
		Profiles:  []Profile{},
	}
}

func defaultProgress(now time.Time) Progress {
	return Progress{
		StartDate:         now.Format("2006-01-02"),
		TrackingSince:     now.Format("2006-01-02"),
		History:           map[string]map[string]HistoryEntry{},
		Completions:       map[string]SessionCompletion{},
		PausedSessions:    map[string]PausedSession{},
		DismissedSessions: map[string]DismissedSession{},
	}
}

func ensureConfigMetadata(config *Config) bool {
	changed := false
	for profileIndex := range config.Profiles {
		profile := &config.Profiles[profileIndex]
		if strings.TrimSpace(profile.ID) == "" {
			profile.ID = stableID("profile", profile.Name, profileIndex)
			changed = true
		} else if normalized := readableID(profile.ID); normalized != profile.ID {
			profile.ID = normalized
			changed = true
		}
		if strings.TrimSpace(profile.PersonID) == "" {
			profile.PersonID = profile.ID
			changed = true
		} else if normalized := readableID(profile.PersonID); normalized != profile.PersonID {
			profile.PersonID = normalized
			changed = true
		}
		if strings.TrimSpace(profile.RunID) == "" {
			profile.RunID = newRunID()
			changed = true
		}
		if strings.TrimSpace(profile.StartDate) == "" {
			profile.StartDate = config.StartDate
			changed = true
		}
		for phaseIndex := range profile.Phases {
			phase := &profile.Phases[phaseIndex]
			if strings.TrimSpace(phase.ID) == "" {
				phase.ID = stableID("phase", profile.ID+"|"+phase.Name, phaseIndex)
				changed = true
			} else if normalized := readableID(phase.ID); normalized != phase.ID {
				phase.ID = normalized
				changed = true
			}
			if ensureSchedulingMetadata(&phase.Scheduling, phase.Program) {
				changed = true
			}
			for dayName, daily := range phase.Schedule {
				if ensureSchedulingMetadata(&daily.Scheduling, daily.Frequency) {
					phase.Schedule[dayName] = daily
					changed = true
				}
			}
		}
	}
	return changed
}

func newRunID() string {
	buffer := make([]byte, 8)
	if _, err := cryptorand.Read(buffer); err == nil {
		return "run_" + hex.EncodeToString(buffer)
	}
	return fmt.Sprintf("run_%d", time.Now().UnixNano())
}

func newPersonID(name string) string {
	base := readableID(name)
	if base == "" {
		base = "osoba"
	}
	buffer := make([]byte, 3)
	if _, err := cryptorand.Read(buffer); err == nil {
		return base + "_" + hex.EncodeToString(buffer)
	}
	return fmt.Sprintf("%s_%d", base, time.Now().UnixNano())
}

func ensurePersonStore(store *PersonStore, config Config, archive ArchiveStore) bool {
	changed := false
	if store.SchemaVersion != 1 {
		store.SchemaVersion = 1
		changed = true
	}
	known := map[string]int{}
	for index := range store.Persons {
		person := &store.Persons[index]
		normalized := readableID(person.ID)
		if normalized == "" {
			normalized = newPersonID(person.Name)
		}
		if normalized != person.ID {
			person.ID = normalized
			changed = true
		}
		known[person.ID] = index
	}
	addProfile := func(profile Profile, active bool) {
		personID := readableID(profile.PersonID)
		if personID == "" {
			personID = readableID(profile.ID)
		}
		if personID == "" {
			personID = newPersonID(profile.Name)
		}
		if index, exists := known[personID]; exists {
			if active && !store.Persons[index].Active {
				store.Persons[index].Active = true
				changed = true
			}
			if strings.TrimSpace(store.Persons[index].Name) == "" && strings.TrimSpace(profile.Name) != "" {
				store.Persons[index].Name = strings.TrimSpace(profile.Name)
				changed = true
			}
			return
		}
		known[personID] = len(store.Persons)
		store.Persons = append(store.Persons, Person{ID: personID, Name: strings.TrimSpace(profile.Name), Active: active})
		changed = true
	}
	for _, profile := range config.Profiles {
		addProfile(profile, true)
	}
	for _, archived := range archive.Profiles {
		addProfile(archived.Profile, false)
	}
	sort.SliceStable(store.Persons, func(i, j int) bool {
		if store.Persons[i].Active != store.Persons[j].Active {
			return store.Persons[i].Active
		}
		return strings.ToLower(store.Persons[i].Name) < strings.ToLower(store.Persons[j].Name)
	})
	return changed
}

func validatePersonStore(store *PersonStore) error {
	store.SchemaVersion = 1
	seen := map[string]bool{}
	for index := range store.Persons {
		person := &store.Persons[index]
		person.ID = readableID(person.ID)
		person.Name = strings.TrimSpace(person.Name)
		if person.ID == "" {
			return fmt.Errorf("osoba %d nie ma identyfikatora", index+1)
		}
		if person.Name == "" {
			return fmt.Errorf("osoba %q nie ma nazwy", person.ID)
		}
		if seen[person.ID] {
			return fmt.Errorf("identyfikator osoby %q występuje więcej niż raz", person.ID)
		}
		seen[person.ID] = true
	}
	return nil
}

func ensureSchedulingMetadata(scheduling *SessionScheduling, program string) bool {
	if strings.TrimSpace(scheduling.ProgramID) != "" {
		normalized := readableID(scheduling.ProgramID)
		if normalized != scheduling.ProgramID {
			scheduling.ProgramID = normalized
			return true
		}
		return false
	}
	if strings.TrimSpace(program) == "" {
		return false
	}
	scheduling.ProgramID = readableID(program)
	return true
}

func stableID(prefix, value string, index int) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(value)) + fmt.Sprintf("|%d", index)))
	return fmt.Sprintf("%s_%x", prefix, sum[:5])
}

func readableID(value string) string {
	var builder strings.Builder
	lastSeparator := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			builder.WriteRune(character)
			lastSeparator = false
		} else if !lastSeparator && builder.Len() > 0 {
			builder.WriteByte('_')
			lastSeparator = true
		}
		if builder.Len() >= 40 {
			break
		}
	}
	return strings.Trim(builder.String(), "_")
}

type programPolicy struct {
	Context    string
	Scheduling SessionScheduling
}

func validWeekdayKey(value string) bool {
	switch value {
	case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday":
		return true
	default:
		return false
	}
}

func validateProgramPolicies(profileName string, policies []programPolicy) error {
	byID := make(map[string]programPolicy, len(policies))
	for _, policy := range policies {
		id := policy.Scheduling.ProgramID
		if id == "" {
			return fmt.Errorf("profil %q, %s: program nie ma identyfikatora", profileName, policy.Context)
		}
		if previous, exists := byID[id]; exists {
			if previous.Scheduling.MinGapMinutes != policy.Scheduling.MinGapMinutes ||
				previous.Scheduling.CooldownAfterMinutes != policy.Scheduling.CooldownAfterMinutes ||
				!sameProgramSet(previous.Scheduling.SameDayWith, policy.Scheduling.SameDayWith) {
				return fmt.Errorf("profil %q: wszystkie wystąpienia programu %q muszą mieć ten sam odstęp, cooldown i zgodność", profileName, id)
			}
			continue
		}
		byID[id] = policy
	}
	for id, policy := range byID {
		for _, targetID := range policy.Scheduling.SameDayWith {
			target, exists := byID[targetID]
			if !exists {
				return fmt.Errorf("profil %q: program %q wskazuje nieistniejący program %q", profileName, id, targetID)
			}
			if !containsProgram(target.Scheduling.SameDayWith, id) {
				return fmt.Errorf("profil %q: zgodność %q i %q musi być wpisana po obu stronach", profileName, id, targetID)
			}
		}
	}
	return nil
}

func sameProgramSet(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for _, value := range first {
		if !containsProgram(second, value) {
			return false
		}
	}
	return true
}

// validationMode rozdziela dwa zupełnie różne konteksty walidacji.
//
// Walidacje STRUKTURALNE (puste ID, nieznany tryb, brak programu, odwołania do
// nieistniejących programów) są twardymi błędami ZAWSZE — bez nich runtime i tak
// nie ruszy. Tryb dotyczy wyłącznie walidacji SPÓJNOŚCIOWYCH (krzyżowych), gdzie
// dane są kompletne i uruchamialne, tylko plan jest wewnętrznie sprzeczny.
//
// validationModeLoad: dane już leżą na dysku. Odrzucenie ich zablokowałoby start
// aplikacji, a użytkownik nie miałby jak ich poprawić — edytor jest w tej samej
// aplikacji. Dlatego sprzeczność zwracamy jako ostrzeżenie, nie błąd.
//
// validationModeSave: użytkownik dopiero wprowadza konfigurację i ma gdzie ją
// naprawić. Sprzeczność pozostaje twardym błędem, żeby nowych nie dało się dodać.
type validationMode int

const (
	validationModeLoad validationMode = iota
	validationModeSave
)

func validateConfig(config *Config, mode validationMode) ([]string, error) {
	var warnings []string
	ensureConfigMetadata(config)
	seenNames := map[string]bool{}
	seenProfileIDs := map[string]bool{}
	for profileIndex := range config.Profiles {
		profile := &config.Profiles[profileIndex]
		profile.Name = strings.TrimSpace(profile.Name)
		if profile.Name == "" {
			return nil, errors.New("każdy profil musi mieć nazwę")
		}
		key := strings.ToLower(profile.Name)
		if seenNames[key] {
			return nil, fmt.Errorf("profil %q występuje więcej niż raz", profile.Name)
		}
		seenNames[key] = true
		if profile.ID == "" || seenProfileIDs[profile.ID] {
			return nil, fmt.Errorf("profil %q ma pusty lub powtórzony identyfikator %q", profile.Name, profile.ID)
		}
		seenProfileIDs[profile.ID] = true
		seenPhaseIDs := map[string]bool{}
		programPolicies := make([]programPolicy, 0)

		for phaseIndex := range profile.Phases {
			phase := &profile.Phases[phaseIndex]
			phase.Name = strings.TrimSpace(phase.Name)
			if phase.ID == "" || seenPhaseIDs[phase.ID] {
				return nil, fmt.Errorf("profil %q ma pusty lub powtórzony identyfikator fazy %q", profile.Name, phase.ID)
			}
			seenPhaseIDs[phase.ID] = true
			if phase.Name == "" {
				return nil, fmt.Errorf("faza w profilu %q musi mieć nazwę", profile.Name)
			}
			if phase.DurationDays < 1 {
				return nil, fmt.Errorf("faza %q musi trwać przynajmniej 1 dzień", phase.Name)
			}
			if phase.Mode != "interval" && phase.Mode != "weekly" {
				return nil, fmt.Errorf("faza %q ma nieznany tryb", phase.Name)
			}
			if phase.IntervalGap < 0 {
				phase.IntervalGap = 0
			}
			if phase.Schedule == nil {
				phase.Schedule = map[string]DailyPlan{}
			}
			if err := validateScheduling(&phase.Scheduling, phase.Name); err != nil {
				return nil, err
			}
			if phase.Mode == "interval" {
				if strings.TrimSpace(phase.Program) == "" {
					return nil, fmt.Errorf("faza %q nie ma programu", phase.Name)
				}
				// Spójność planu jest twarda tylko przy zapisie. Przy wczytywaniu
				// zamienia się w ostrzeżenie — inaczej dane już leżące na dysku
				// zablokowałyby start aplikacji, czyli jedyne miejsce, w którym
				// użytkownik mógłby je naprawić.
				for _, issue := range intervalConsistencyIssues(profile.Name, *phase) {
					if mode == validationModeSave {
						return nil, errors.New(issue.detail)
					}
					warnings = append(warnings, issue.brief)
				}
				programPolicies = append(programPolicies, programPolicy{Context: phase.Name, Scheduling: phase.Scheduling})
			}
			if len(phase.DeviceSteps) > 0 {
				if err := validateDeviceSteps(phase.DeviceSteps); err != nil {
					return nil, fmt.Errorf("faza %q: %w", phase.Name, err)
				}
			}
			for dayName, daily := range phase.Schedule {
				if !validWeekdayKey(dayName) {
					return nil, fmt.Errorf("faza %q ma nieznany dzień tygodnia %q", phase.Name, dayName)
				}
				if err := validateScheduling(&daily.Scheduling, phase.Name+" / "+dayName); err != nil {
					return nil, err
				}
				// KLUCZOWE: `daily` to kopia wartości z mapy. Bez tego zapisu
				// normalizacja (readableID, dedup, usunięcie self-referencji)
				// ginęła, a runtime porównywał "program_b" z wpisanym "Program B".
				phase.Schedule[dayName] = daily
				// Polityki programów zbieramy tylko z trybu, który planForDate
				// naprawdę czyta. W fazie interwałowej mapa Schedule jest ignorowana,
				// więc nie może generować błędów walidacji dla nieużywanych danych.
				if phase.Mode == "weekly" && strings.TrimSpace(daily.Frequency) != "" {
					programPolicies = append(programPolicies, programPolicy{Context: phase.Name + " / " + dayName, Scheduling: daily.Scheduling})
				}
				if len(daily.DeviceSteps) == 0 {
					continue
				}
				if err := validateDeviceSteps(daily.DeviceSteps); err != nil {
					return nil, fmt.Errorf("faza %q, dzień %s: %w", phase.Name, dayName, err)
				}
			}
		}
		if err := validateProgramPolicies(profile.Name, programPolicies); err != nil {
			return nil, err
		}
	}
	return warnings, nil
}

// intervalConsistencyIssue opisuje jedną sprzeczność planu w dwóch wariantach:
// detail trafia do błędu zapisu (pełna instrukcja naprawy), brief do ostrzeżenia
// przy wczytywaniu, gdzie liczy się jedno czytelne zdanie w banerze.
type intervalConsistencyIssue struct {
	detail string
	brief  string
}

// intervalConsistencyIssues wyłapuje konfiguracje wewnętrznie sprzeczne:
// plan generuje terminy, których minimalny odstęp nigdy nie pozwoli dotrzymać,
// więc każda kolejna sesja od razu wpada w zaległości.
//
// Faza interwałowa z interval_gap = N stawia sesje co N+1 dni kalendarzowych,
// czyli co (N+1)*1440 minut. Odstęp większy od tego okresu jest niewykonalny.
//
// Próg celowo pozostaje ostry (>), a nie >=. Przy min_gap == okres plan jest
// wykonalny, tylko wymaga punktualności co do minuty — to informacja dla
// użytkownika (ostrzeżenie inline w edytorze), a nie powód do blokady danych.
func intervalConsistencyIssues(profileName string, phase Phase) []intervalConsistencyIssue {
	gapDays := phase.IntervalGap
	if gapDays < 0 {
		gapDays = 0
	}
	periodMinutes := (gapDays + 1) * 24 * 60
	// Faza mieszcząca tylko jeden termin nie może być sprzeczna — nie ma "następnej" sesji.
	multipleSessions := phase.DurationDays > gapDays+1
	if !multipleSessions {
		return nil
	}
	program := valueOr(phase.Program, phase.Scheduling.ProgramID)
	issues := make([]intervalConsistencyIssue, 0, 2)
	if phase.Scheduling.MinGapMinutes > periodMinutes {
		issues = append(issues, intervalConsistencyIssue{
			detail: fmt.Sprintf(
				"profil %q, faza %q, program %q: minimalny odstęp %d min jest większy niż odstęp między zaplanowanymi sesjami (%d min co %d dni) — takiego planu nigdy nie da się wykonać na czas; zmniejsz minimalny odstęp albo zwiększ przerwę między sesjami",
				profileName, phase.Name, program, phase.Scheduling.MinGapMinutes, periodMinutes, gapDays+1),
			brief: fmt.Sprintf(
				"Faza %q profilu %q ma niewykonalny plan: minimalny odstęp %d min przekracza okres %d min — popraw w edytorze",
				phase.Name, profileName, phase.Scheduling.MinGapMinutes, periodMinutes),
		})
	}
	repetitions := phase.Scheduling.Repetitions
	if repetitions > 1 && phase.Scheduling.BreakBetweenMinutes > 0 {
		spanMinutes := (repetitions - 1) * phase.Scheduling.BreakBetweenMinutes
		if spanMinutes > periodMinutes {
			issues = append(issues, intervalConsistencyIssue{
				detail: fmt.Sprintf(
					"profil %q, faza %q, program %q: %d powtórzeń z przerwą %d min zajmuje %d min, a kolejna sesja jest planowana już po %d min — powtórzenia nie zmieszczą się przed następnym terminem",
					profileName, phase.Name, program, repetitions, phase.Scheduling.BreakBetweenMinutes, spanMinutes, periodMinutes),
				brief: fmt.Sprintf(
					"Faza %q profilu %q ma niewykonalny plan: %d powtórzeń zajmuje %d min i nie mieści się w okresie %d min — popraw w edytorze",
					phase.Name, profileName, repetitions, spanMinutes, periodMinutes),
			})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return issues
}

func validateScheduling(scheduling *SessionScheduling, context string) error {
	scheduling.ProgramID = readableID(scheduling.ProgramID)
	if scheduling.Repetitions < 0 || scheduling.Repetitions > 12 {
		return fmt.Errorf("%s: liczba powtórzeń musi mieścić się w zakresie 1–12", context)
	}
	if scheduling.BreakBetweenMinutes < 0 || scheduling.BreakBetweenMinutes > 24*60 {
		return fmt.Errorf("%s: przerwa między powtórzeniami jest poza zakresem 0–1440 min", context)
	}
	if scheduling.MinGapMinutes < 0 || scheduling.MinGapMinutes > 30*24*60 {
		return fmt.Errorf("%s: minimalny odstęp jest poza zakresem 0–43200 min", context)
	}
	if scheduling.CooldownAfterMinutes < 0 || scheduling.CooldownAfterMinutes > 30*24*60 {
		return fmt.Errorf("%s: odstęp po sesji jest poza zakresem 0–43200 min", context)
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(scheduling.SameDayWith))
	for _, value := range scheduling.SameDayWith {
		value = readableID(value)
		if value != "" && value != scheduling.ProgramID && !seen[value] {
			seen[value] = true
			clean = append(clean, value)
		}
	}
	scheduling.SameDayWith = clean
	return nil
}
