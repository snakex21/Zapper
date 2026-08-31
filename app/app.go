package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Zmienna pozwala zbudować starszą wersję wyłącznie do testu aktualizatora
// przez -X main.appVersion=x.y.z. Zwykłe i release'owe buildy używają wartości poniżej.
var appVersion = "10.4.2"

// maxManualRunRecords ogranicza rozrost dziennika trybu ręcznego.
const maxManualRunRecords = 2000

type Application struct {
	mu                 sync.Mutex
	appDirectory       string
	profilesPath       string
	personsPath        string
	progressPath       string
	legacyProgressPath string
	archivePath        string
	legacyArchivePath  string
	manualRunsPath     string
	config             Config
	progress           Progress
	archive            ArchiveStore
	persons            PersonStore
	manualRuns         ManualRunStore
	frequencies        []FrequencyEntry
	loadWarnings       []string
	now                func() time.Time
}

// LoadWarnings zwraca ostrzeżenia spójnościowe wykryte przy wczytywaniu danych.
// Nie zablokowały startu, ale użytkownik musi się o nich dowiedzieć.
func (a *Application) LoadWarnings() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.loadWarnings...)
}

func NewApplication(appDirectory string) (*Application, error) {
	now := time.Now()
	dataDirectory := filepath.Join(appDirectory, "data")
	if err := prepareDataDirectory(appDirectory, dataDirectory); err != nil {
		return nil, err
	}
	application := &Application{
		appDirectory:       appDirectory,
		profilesPath:       filepath.Join(dataDirectory, "profiles.json"),
		personsPath:        filepath.Join(dataDirectory, "persons.json"),
		progressPath:       filepath.Join(dataDirectory, "progress"),
		legacyProgressPath: filepath.Join(dataDirectory, "progress.json"),
		archivePath:        filepath.Join(dataDirectory, "archive"),
		legacyArchivePath:  filepath.Join(dataDirectory, "archive.json"),
		manualRunsPath:     filepath.Join(dataDirectory, "manual_runs.json"),
		frequencies:        loadFrequencyDatabase(),
		now:                time.Now,
	}

	if err := loadJSON(application.profilesPath, &application.config); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		application.config = defaultConfig(now)
		if err := saveJSONAtomic(application.profilesPath, application.config); err != nil {
			return nil, err
		}
	}
	configChanged := ensureConfigMetadata(&application.config)
	// Wczytywanie NIE odrzuca danych za sprzeczny plan — użytkownik siedziałby
	// wtedy przed aplikacją, która się nie uruchamia, bez dostępu do edytora.
	// Sprzeczności wracają jako ostrzeżenia i lądują w banerze oraz errors.log.
	loadWarnings, err := validateConfig(&application.config, validationModeLoad)
	if err != nil {
		return nil, fmt.Errorf("nieprawidłowy profiles.json: %w", err)
	}
	application.loadWarnings = loadWarnings
	if configChanged {
		if err := saveJSONAtomic(application.profilesPath, application.config); err != nil {
			return nil, err
		}
	}
	migratedProgress := false
	if err := loadJSON(application.legacyProgressPath, &application.progress); err == nil {
		migratedProgress = true
	} else if os.IsNotExist(err) {
		loadedProgress, loadErr := loadActiveProgress(application.progressPath, application.config, now)
		if loadErr != nil {
			return nil, loadErr
		}
		application.progress = loadedProgress
	} else {
		return nil, err
	}
	if application.progress.History == nil {
		application.progress.History = map[string]map[string]HistoryEntry{}
	}
	progressChanged := false
	if application.progress.Completions == nil {
		application.progress.Completions = map[string]SessionCompletion{}
		progressChanged = true
	}
	if application.progress.PausedSessions == nil {
		application.progress.PausedSessions = map[string]PausedSession{}
		progressChanged = true
	}
	if application.progress.DismissedSessions == nil {
		application.progress.DismissedSessions = map[string]DismissedSession{}
		progressChanged = true
	}
	if application.progress.TrackingSince == "" {
		application.progress.TrackingSince = dateOnly(now).Format("2006-01-02")
		progressChanged = true
	}
	if progressChanged || migratedProgress {
		if err := saveActiveProgress(application.progressPath, application.config, application.progress); err != nil {
			return nil, err
		}
	}
	if migratedProgress {
		if err := preserveMigratedFile(application.legacyProgressPath); err != nil {
			return nil, err
		}
	}
	loadedArchive, err := loadArchiveFolders(application.archivePath)
	if err != nil {
		return nil, err
	}
	application.archive = loadedArchive
	var legacyArchive ArchiveStore
	if err := loadJSON(application.legacyArchivePath, &legacyArchive); err == nil {
		known := map[string]bool{}
		for _, archived := range application.archive.Profiles {
			known[archived.ArchiveID] = true
		}
		for _, archived := range legacyArchive.Profiles {
			if known[archived.ArchiveID] {
				continue
			}
			archivedProgress := progressForProfile(application.progress, archived.Profile)
			if err := saveArchiveFolder(application.archivePath, archived, archivedProgress); err != nil {
				return nil, err
			}
			archived.Progress = &archivedProgress
			application.archive.Profiles = append(application.archive.Profiles, archived)
		}
		if err := preserveMigratedFile(application.legacyArchivePath); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	personsChanged := false
	if err := loadJSON(application.personsPath, &application.persons); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		application.persons = PersonStore{SchemaVersion: 1, Persons: []Person{}}
		personsChanged = true
	}
	if ensurePersonStore(&application.persons, application.config, application.archive) {
		personsChanged = true
	}
	if err := validatePersonStore(&application.persons); err != nil {
		return nil, fmt.Errorf("nieprawidłowy persons.json: %w", err)
	}
	if personsChanged {
		if err := saveJSONAtomic(application.personsPath, application.persons); err != nil {
			return nil, err
		}
	}
	manualRuns, err := loadManualRuns(application.manualRunsPath)
	if err != nil {
		return nil, err
	}
	application.manualRuns = manualRuns
	return application, nil
}

func (a *Application) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snapshotLocked(30)
}

// validateRequestedPersonID sprawdza ID podane ręcznie przez użytkownika.
// Zwraca ID znormalizowane do małych liter; dozwolone są wyłącznie [a-z0-9_-].
func validateRequestedPersonID(requestedID string) (string, error) {
	trimmed := strings.TrimSpace(requestedID)
	if trimmed == "" {
		return "", errors.New("podane ID jest puste")
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return "", fmt.Errorf("ID %q nie może zawierać spacji", requestedID)
	}
	normalized := strings.ToLower(trimmed)
	for _, character := range normalized {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '-' {
			continue
		}
		return "", fmt.Errorf("ID %q może zawierać tylko litery a-z, cyfry, podkreślenie i myślnik", requestedID)
	}
	if len(normalized) > 40 {
		return "", fmt.Errorf("ID %q jest za długie (maksymalnie 40 znaków)", requestedID)
	}
	return normalized, nil
}

// AddPerson dodaje osobę. Gdy requestedID jest puste, ID jest generowane jak dotąd
// (nazwa + losowy sufiks) i działa reaktywacja istniejącej osoby po nazwie.
// Gdy requestedID jest podane, jest walidowane i musi być unikalne.
func (a *Application) AddPerson(name string, requestedID string) (PersonMutationResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		return PersonMutationResult{}, errors.New("podaj nazwę osoby")
	}
	explicitID := ""
	if strings.TrimSpace(requestedID) != "" {
		validated, err := validateRequestedPersonID(requestedID)
		if err != nil {
			return PersonMutationResult{}, err
		}
		explicitID = validated
		for index := range a.persons.Persons {
			if a.persons.Persons[index].ID != explicitID {
				continue
			}
			// To samo ID i ta sama nazwa: traktujemy jak reaktywację, nie jak kolizję.
			if strings.EqualFold(strings.TrimSpace(a.persons.Persons[index].Name), name) {
				if !a.persons.Persons[index].Active {
					a.persons.Persons[index].Active = true
					if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
						return PersonMutationResult{}, err
					}
				}
				return PersonMutationResult{Snapshot: a.snapshotLocked(30), PersonID: explicitID}, nil
			}
			return PersonMutationResult{}, fmt.Errorf("ID %q jest już zajęte przez osobę %q", explicitID, a.persons.Persons[index].Name)
		}
	}
	for index := range a.persons.Persons {
		if !strings.EqualFold(strings.TrimSpace(a.persons.Persons[index].Name), name) {
			continue
		}
		if explicitID != "" && a.persons.Persons[index].ID != explicitID {
			return PersonMutationResult{}, fmt.Errorf("osoba %q już istnieje pod ID %q — użyj tego ID albo innej nazwy", name, a.persons.Persons[index].ID)
		}
		if !a.persons.Persons[index].Active {
			a.persons.Persons[index].Active = true
			if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
				return PersonMutationResult{}, err
			}
		}
		return PersonMutationResult{Snapshot: a.snapshotLocked(30), PersonID: a.persons.Persons[index].ID}, nil
	}
	newID := explicitID
	if newID == "" {
		newID = newPersonID(name)
	}
	person := Person{ID: newID, Name: name, Active: true}
	a.persons.Persons = append(a.persons.Persons, person)
	if err := validatePersonStore(&a.persons); err != nil {
		return PersonMutationResult{}, err
	}
	if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
		return PersonMutationResult{}, err
	}
	return PersonMutationResult{Snapshot: a.snapshotLocked(30), PersonID: person.ID}, nil
}

func (a *Application) UpdatePerson(person Person) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	person.ID = readableID(person.ID)
	person.Name = strings.TrimSpace(person.Name)
	index := -1
	for candidate := range a.persons.Persons {
		if a.persons.Persons[candidate].ID == person.ID {
			index = candidate
			break
		}
	}
	if index < 0 {
		return Snapshot{}, errors.New("nie znaleziono osoby")
	}
	if !person.Active {
		for _, profile := range a.config.Profiles {
			if profilePersonID(profile) == person.ID {
				return Snapshot{}, errors.New("nie można ukryć osoby z aktywnym programem")
			}
		}
	}
	a.persons.Persons[index] = person
	if err := validatePersonStore(&a.persons); err != nil {
		return Snapshot{}, err
	}
	for profileIndex := range a.config.Profiles {
		if profilePersonID(a.config.Profiles[profileIndex]) == person.ID {
			a.config.Profiles[profileIndex].Name = person.Name
			a.config.Profiles[profileIndex].PersonID = person.ID
		}
	}
	if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
		return Snapshot{}, err
	}
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

// DeletePerson usuwa osobę całkowicie: wpis z persons.json oraz wszystkie jej
// zarchiwizowane programy (foldery archiwum). Bez tego osoba po restarcie
// wracałaby na listę jako nieaktywna, bo ensurePersonStore odtwarza osoby
// z archiwum.
func (a *Application) DeletePerson(personID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	personID = readableID(personID)
	if _, exists := a.personByIDLocked(personID); !exists {
		return Snapshot{}, errors.New("nie znaleziono osoby")
	}
	for _, profile := range a.config.Profiles {
		if profilePersonID(profile) == personID {
			return Snapshot{}, errors.New("najpierw zakończ lub usuń program tej osoby")
		}
	}
	keptPersons := make([]Person, 0, len(a.persons.Persons))
	for _, person := range a.persons.Persons {
		if person.ID != personID {
			keptPersons = append(keptPersons, person)
		}
	}
	a.persons.Persons = keptPersons
	keptArchive := make([]ArchivedProfile, 0, len(a.archive.Profiles))
	for _, archived := range a.archive.Profiles {
		if profilePersonID(archived.Profile) != personID {
			keptArchive = append(keptArchive, archived)
			continue
		}
		if err := os.RemoveAll(filepath.Join(a.archivePath, archived.ArchiveID)); err != nil {
			return Snapshot{}, err
		}
	}
	a.archive.Profiles = keptArchive
	a.progress = progressWithoutPerson(a.progress, personID)
	if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) CreateProfileForPerson(personID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	person, exists := a.personByIDLocked(personID)
	if !exists {
		return Snapshot{}, errors.New("nie znaleziono osoby")
	}
	for _, profile := range a.config.Profiles {
		if profilePersonID(profile) == person.ID {
			return Snapshot{}, errors.New("ta osoba ma już aktywny program")
		}
	}
	profile := Profile{ID: person.ID, PersonID: person.ID, RunID: newRunID(), StartDate: a.now().Format("2006-01-02"), Name: person.Name, Phases: []Phase{}}
	a.config.Profiles = append(a.config.Profiles, profile)
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) personByIDLocked(personID string) (Person, bool) {
	personID = readableID(personID)
	for _, person := range a.persons.Persons {
		if person.ID == personID {
			return person, true
		}
	}
	return Person{}, false
}

func profilePersonID(profile Profile) string {
	if value := readableID(profile.PersonID); value != "" {
		return value
	}
	return readableID(profile.ID)
}

func (a *Application) SaveConfig(config Config) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if config.StartDate == "" {
		config.StartDate = a.config.StartDate
	}
	ensureConfigMetadata(&config)
	seenPersons := map[string]bool{}
	personsChanged := false
	for index := range config.Profiles {
		personID := profilePersonID(config.Profiles[index])
		person, exists := a.personByIDLocked(personID)
		if !exists {
			person = Person{ID: personID, Name: strings.TrimSpace(config.Profiles[index].Name), Active: true}
			if person.ID == "" {
				person.ID = newPersonID(person.Name)
			}
			a.persons.Persons = append(a.persons.Persons, person)
			personsChanged = true
		}
		if seenPersons[person.ID] {
			return Snapshot{}, fmt.Errorf("osoba %q ma więcej niż jeden aktywny program", person.Name)
		}
		seenPersons[person.ID] = true
		config.Profiles[index].PersonID = person.ID
		config.Profiles[index].Name = person.Name
		if strings.TrimSpace(config.Profiles[index].StartDate) == "" {
			config.Profiles[index].StartDate = a.now().Format("2006-01-02")
		}
	}
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		return Snapshot{}, err
	}
	if personsChanged {
		if err := validatePersonStore(&a.persons); err != nil {
			return Snapshot{}, err
		}
		if err := saveJSONAtomic(a.personsPath, a.persons); err != nil {
			return Snapshot{}, err
		}
	}
	if err := saveJSONAtomic(a.profilesPath, config); err != nil {
		return Snapshot{}, err
	}
	a.config = config
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) SetDone(profileName string, done bool) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	profileName = strings.TrimSpace(profileName)
	plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
	for _, plan := range plans {
		if plan.ProfileName != profileName || plan.Status != "session" || plan.Done == done {
			continue
		}
		return a.setSessionDoneLocked(plan, done, false)
	}
	return Snapshot{}, fmt.Errorf("nie znaleziono %s sesji profilu %q", map[bool]string{true: "oczekującej", false: "wykonanej"}[done], profileName)
}

func (a *Application) TodayPlan(profileName string) (DayPlan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
	for _, plan := range plans {
		if plan.ProfileName == strings.TrimSpace(profileName) && plan.Status == "session" && !plan.Done {
			return plan, nil
		}
	}
	return DayPlan{}, fmt.Errorf("profil %q nie ma oczekującej sesji", profileName)
}

func (a *Application) TodayPlanBySession(sessionID string) (DayPlan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.todayPlanBySessionLocked(sessionID)
}

func (a *Application) todayPlanBySessionLocked(sessionID string) (DayPlan, error) {
	plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
	for _, plan := range plans {
		if plan.SessionID == strings.TrimSpace(sessionID) {
			return plan, nil
		}
	}
	return DayPlan{}, errors.New("nie znaleziono tej sesji w bieżącym planie")
}

func (a *Application) SetSessionDone(sessionID string, done bool) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	plan, err := a.todayPlanBySessionLocked(sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	return a.setSessionDoneLocked(plan, done, false)
}

// CompleteDeviceSession zapisuje zabieg, który FIZYCZNIE się odbył (płytka zgłosiła DONE).
// Świadomie pomija kontrolę Available: skoro sesja przebiegła na urządzeniu, odmowa
// zapisu byłaby gorsza niż zapis — zabieg zniknąłby z dziennika, a odstęp po nim
// nigdy nie zacząłby biec. Kontrola dostępności należy do momentu STARTU sesji
// (apiDeviceStartSession), nie do momentu jej zakończenia.
func (a *Application) CompleteDeviceSession(sessionID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	plan, err := a.todayPlanBySessionLocked(sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	return a.setSessionDoneLocked(plan, true, true)
}

func (a *Application) setSessionDoneLocked(plan DayPlan, done bool, force bool) (Snapshot, error) {
	if done && plan.Done {
		return a.snapshotLocked(30), nil
	}
	if done && !plan.Available && !force {
		return Snapshot{}, fmt.Errorf("sesja jest jeszcze niedostępna: %s", plan.BlockedReason)
	}
	if a.progress.Completions == nil {
		a.progress.Completions = map[string]SessionCompletion{}
	}
	completedAt := a.now().Format(time.RFC3339)
	if done {
		// Przebieg wstrzymany i dokończony później jest w historii oznaczany jako dzielony,
		// razem z momentem pierwszej pauzy — inaczej wznowienie po tygodniu wyglądałoby
		// jak jedno ciągłe wykonanie.
		split := false
		firstStartedAt := ""
		if paused, exists := a.progress.PausedSessions[plan.SessionID]; exists {
			split = true
			firstStartedAt = valueOr(paused.FirstPausedAt, paused.PausedAt)
		}
		a.progress.Completions[plan.SessionID] = SessionCompletion{
			Split:          split,
			FirstStartedAt: firstStartedAt,
			SessionID:      plan.SessionID,
			SessionGroupID: plan.SessionGroupID,
			ProfileID:      plan.ProfileID,
			PersonID:       plan.PersonID,
			RunID:          plan.RunID,
			ProfileName:    plan.ProfileName,
			ProgramID:      plan.Scheduling.ProgramID,
			PlannedDate:    plan.PlannedDate,
			CompletedAt:    completedAt,
			Therapy:        plan.Program,
			Time:           plan.Time,
			Repetitions:    plan.Repetitions,
			Phase:          plan.PhaseName,
			Notes:          plan.Note,
			Scheduling:     plan.Scheduling,
		}
		delete(a.progress.PausedSessions, plan.SessionID)
	} else {
		delete(a.progress.Completions, plan.SessionID)
	}
	a.syncLegacyHistory(plan, done, completedAt)
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	if done {
		if err := a.autoArchiveCompletedProfilesLocked(); err != nil {
			return Snapshot{}, err
		}
	}
	return a.snapshotLocked(30), nil
}

// SavePausedSession zapisuje pauzę sesji. Gdy recordPartial jest prawdziwe
// (przycisk Zatrzymaj), dopisuje też ślad częściowego wykonania do historii —
// sesja nadal czeka na dokończenie, ale wiadomo, ile już zrobiono i ile zostało.
func (a *Application) SavePausedSession(pause DevicePauseState, recordPartial bool) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	plan, err := a.todayPlanBySessionLocked(pause.SessionID)
	if err != nil {
		return Snapshot{}, err
	}
	if len(pause.RemainingSteps) == 0 || pause.RemainingSeconds == 0 {
		return Snapshot{}, errors.New("pauza nie zawiera pozostałego czasu")
	}
	if a.progress.PausedSessions == nil {
		a.progress.PausedSessions = map[string]PausedSession{}
	}
	firstPausedAt := a.now().Format(time.RFC3339)
	if previous, exists := a.progress.PausedSessions[plan.SessionID]; exists {
		firstPausedAt = valueOr(previous.FirstPausedAt, previous.PausedAt)
	}
	a.progress.PausedSessions[plan.SessionID] = PausedSession{
		FirstPausedAt:    firstPausedAt,
		SessionID:        plan.SessionID,
		ProfileID:        plan.ProfileID,
		PersonID:         plan.PersonID,
		RunID:            plan.RunID,
		ProfileName:      plan.ProfileName,
		PausedAt:         a.now().Format(time.RFC3339),
		RemainingSteps:   append([]DeviceStep(nil), pause.RemainingSteps...),
		RemainingSeconds: pause.RemainingSeconds,
		Recorded:         recordPartial,
	}
	if recordPartial {
		var total uint32
		for _, step := range plan.DeviceSteps {
			total += step.DurationSeconds
		}
		done := total
		if pause.RemainingSeconds < total {
			done = total - pause.RemainingSeconds
		}
		if a.progress.PartialRuns == nil {
			a.progress.PartialRuns = []PartialRun{}
		}
		a.progress.PartialRuns = append(a.progress.PartialRuns, PartialRun{
			SessionID:        plan.SessionID,
			ProfileID:        plan.ProfileID,
			PersonID:         plan.PersonID,
			RunID:            plan.RunID,
			ProfileName:      plan.ProfileName,
			ProgramID:        plan.Scheduling.ProgramID,
			Therapy:          plan.Program,
			Phase:            plan.PhaseName,
			PlannedDate:      plan.PlannedDate,
			DoneSeconds:      done,
			TotalSeconds:     total,
			RemainingSeconds: pause.RemainingSeconds,
			RecordedAt:       a.now().Format(time.RFC3339),
		})
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

// DismissOverdueSessions odpuszcza wszystkie zaległe sesje profilu.
//
// To NIE jest oznaczenie ich jako wykonanych — trafiają do osobnej mapy
// DismissedSessions, nie do Completions, więc nie liczą się w historii,
// statystykach ani w kontekście dla AI. Chodzi wyłącznie o domknięcie:
// po miesiącu przerwy profil przestaje wisieć w nieskończoność z kolejką
// zaległości czyszczoną po jednej dziennie i może się normalnie zarchiwizować.
func (a *Application) DismissOverdueSessions(profileID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	profileID = strings.TrimSpace(profileID)
	found := false
	for _, profile := range a.config.Profiles {
		if profile.ID == profileID {
			found = true
			break
		}
	}
	if !found {
		return Snapshot{}, errors.New("nie znaleziono aktywnego profilu")
	}
	if a.progress.DismissedSessions == nil {
		a.progress.DismissedSessions = map[string]DismissedSession{}
	}
	dismissedAt := a.now().Format(time.RFC3339)
	plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
	count := 0
	for _, plan := range plans {
		if plan.ProfileID != profileID || plan.Status != "session" || plan.Done || !plan.Overdue {
			continue
		}
		a.progress.DismissedSessions[plan.SessionID] = DismissedSession{
			SessionID:      plan.SessionID,
			SessionGroupID: plan.SessionGroupID,
			ProfileID:      plan.ProfileID,
			PersonID:       plan.PersonID,
			RunID:          plan.RunID,
			ProfileName:    plan.ProfileName,
			ProgramID:      plan.Scheduling.ProgramID,
			Therapy:        plan.Program,
			Phase:          plan.PhaseName,
			PlannedDate:    plan.PlannedDate,
			DismissedAt:    dismissedAt,
		}
		delete(a.progress.PausedSessions, plan.SessionID)
		count++
	}
	if count == 0 {
		return Snapshot{}, errors.New("ten profil nie ma zaległych sesji do odpuszczenia")
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	if err := a.autoArchiveCompletedProfilesLocked(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

// DismissSessionGroup odpuszcza jeden termin jako całość. Dla serii 3x zapisuje
// trzy odpuszczone części, żeby żadna z nich nie wróciła później do kolejki.
// Nie tworzy wykonania i nie wpływa na odstępy liczone od faktycznych zabiegów.
func (a *Application) DismissSessionGroup(sessionID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	plans, _ := buildOperationalPlans(a.config, a.progress, a.now())
	var target DayPlan
	found := false
	for _, plan := range plans {
		if plan.SessionID == strings.TrimSpace(sessionID) {
			target = plan
			found = true
			break
		}
	}
	if !found || target.Status != "session" {
		return Snapshot{}, errors.New("nie znaleziono tego terminu w bieżącym planie")
	}
	if a.progress.DismissedSessions == nil {
		a.progress.DismissedSessions = map[string]DismissedSession{}
	}
	dismissedAt := a.now().Format(time.RFC3339)
	count := 0
	for _, plan := range plans {
		if plan.SessionGroupID != target.SessionGroupID || plan.Done {
			continue
		}
		a.progress.DismissedSessions[plan.SessionID] = DismissedSession{
			SessionID:      plan.SessionID,
			SessionGroupID: plan.SessionGroupID,
			ProfileID:      plan.ProfileID,
			PersonID:       plan.PersonID,
			RunID:          plan.RunID,
			ProfileName:    plan.ProfileName,
			ProgramID:      plan.Scheduling.ProgramID,
			Therapy:        plan.Program,
			Phase:          plan.PhaseName,
			PlannedDate:    plan.PlannedDate,
			DismissedAt:    dismissedAt,
		}
		delete(a.progress.PausedSessions, plan.SessionID)
		count++
	}
	if count == 0 {
		return Snapshot{}, errors.New("ten termin jest już wykonany albo odpuszczony")
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	if err := a.autoArchiveCompletedProfilesLocked(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

// RecordManualRun zapisuje ślad po ręcznym uruchomieniu płytki.
// Tryb ręczny celowo nie podlega kontroli odstępów — to furtka użytkownika —
// ale musi zostawiać ślad, żeby dało się go zobaczyć w historii i uczciwie
// policzyć odstępy. Rekord trafia do osobnego pliku, nie do postępu profili.
func (a *Application) RecordManualRun(frequencyMilliHz uint64, durationSeconds uint32) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	startedAt := a.now()
	a.manualRuns.Runs = append(a.manualRuns.Runs, ManualRun{
		RunRecordID:      fmt.Sprintf("manual_%d", startedAt.UnixNano()),
		StartedAt:        startedAt.Format(time.RFC3339),
		FrequencyMilliHz: frequencyMilliHz,
		DurationSeconds:  durationSeconds,
		Source:           "manual",
	})
	if len(a.manualRuns.Runs) > maxManualRunRecords {
		a.manualRuns.Runs = a.manualRuns.Runs[len(a.manualRuns.Runs)-maxManualRunRecords:]
	}
	return saveJSONAtomic(a.manualRunsPath, a.manualRuns)
}

func (a *Application) CancelPausedSession(sessionID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sessionID = strings.TrimSpace(sessionID)
	if _, exists := a.progress.PausedSessions[sessionID]; !exists {
		return Snapshot{}, errors.New("ta sesja nie jest wstrzymana")
	}
	delete(a.progress.PausedSessions, sessionID)
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) DiscardPausedSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sessionID = strings.TrimSpace(sessionID)
	if _, exists := a.progress.PausedSessions[sessionID]; !exists {
		return
	}
	delete(a.progress.PausedSessions, sessionID)
	_ = saveActiveProgress(a.progressPath, a.config, a.progress)
}

func (a *Application) syncLegacyHistory(plan DayPlan, done bool, completedAt string) {
	if a.progress.History == nil {
		a.progress.History = map[string]map[string]HistoryEntry{}
	}
	dateKey := dateOnly(a.now()).Format("2006-01-02")
	if a.progress.History[dateKey] == nil {
		a.progress.History[dateKey] = map[string]HistoryEntry{}
	}
	if !done {
		delete(a.progress.History[dateKey], plan.ProfileName)
		return
	}
	a.progress.History[dateKey][plan.ProfileName] = HistoryEntry{
		Done:        true,
		Therapy:     plan.Program,
		Time:        plan.Time,
		Repetitions: plan.Repetitions,
		Phase:       plan.PhaseName,
		Notes:       plan.Note,
		CompletedAt: completedAt,
	}
}

func (a *Application) FinishProfile(profileID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.archiveProfileByIDLocked(profileID, "Zakończony ręcznie")
}

// DeleteProfile usuwa profil z konfiguracji, ale najpierw archiwizuje go razem z
// postępem, żeby historia nie została osierocona.
func (a *Application) DeleteProfile(profileID string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.archiveProfileByIDLocked(profileID, "Usunięty ręcznie")
}

func (a *Application) archiveProfileByIDLocked(profileID string, reason string) (Snapshot, error) {
	profileID = strings.TrimSpace(profileID)
	profileIndex := -1
	for index := range a.config.Profiles {
		if a.config.Profiles[index].ID == profileID {
			profileIndex = index
			break
		}
	}
	if profileIndex < 0 {
		return Snapshot{}, errors.New("nie znaleziono aktywnego profilu")
	}
	profile := a.config.Profiles[profileIndex]
	_, states := buildOperationalPlans(a.config, a.progress, a.now())
	state := ProfileRuntimeStatus{ProfileID: profile.ID, ProfileName: profile.Name}
	for _, candidate := range states {
		if candidate.ProfileID == profile.ID {
			state = candidate
			break
		}
	}
	completedCount := 0
	modernCompletionDates := map[string]bool{}
	for _, completion := range a.progress.Completions {
		if completion.ProfileID == profile.ID {
			completedCount++
			if completedAt, valid := parseTimestamp(completion.CompletedAt); valid {
				modernCompletionDates[dateOnly(completedAt.In(a.now().Location())).Format("2006-01-02")] = true
			}
		}
	}
	for date, entries := range a.progress.History {
		if entry, exists := entries[profile.Name]; exists && entry.Done && !modernCompletionDates[date] {
			completedCount++
		}
	}
	archived := ArchivedProfile{
		ArchiveID:      fmt.Sprintf("archive_%s_%d", profile.ID, a.now().Unix()),
		FinishedAt:     a.now().Format(time.RFC3339),
		Reason:         reason,
		Profile:        profile,
		OverdueCount:   state.OverdueCount,
		ExtensionDays:  state.ExtensionDays,
		CompletedCount: completedCount,
	}
	archivedProgress := progressForProfile(a.progress, profile)
	if err := saveArchiveFolder(a.archivePath, archived, archivedProgress); err != nil {
		return Snapshot{}, err
	}
	archived.Progress = &archivedProgress
	if err := removeActiveProgressFile(a.progressPath, profile); err != nil {
		return Snapshot{}, err
	}
	a.archive.Profiles = append([]ArchivedProfile{archived}, a.archive.Profiles...)
	a.config.Profiles = append(a.config.Profiles[:profileIndex], a.config.Profiles[profileIndex+1:]...)
	a.progress = progressWithoutProfile(a.progress, profile)
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) autoArchiveCompletedProfilesLocked() error {
	if len(a.config.Profiles) == 0 {
		return nil
	}
	now := a.now()
	today := dateOnly(now)
	defaultStartDate := effectiveStartDate(a.config, a.progress, today)
	plans, states := buildOperationalPlans(a.config, a.progress, now)
	stateByID := make(map[string]ProfileRuntimeStatus, len(states))
	for _, state := range states {
		stateByID[state.ProfileID] = state
	}
	pendingByID := map[string]bool{}
	for _, plan := range plans {
		if plan.Status == "session" && !plan.Done {
			pendingByID[plan.ProfileID] = true
		}
	}
	completedIDs := map[string]bool{}
	for _, profile := range a.config.Profiles {
		if len(profile.Phases) == 0 || pendingByID[profile.ID] {
			continue
		}
		if profileScheduleEnded(profile, profileStartDate(profile, defaultStartDate), today) {
			completedIDs[profile.ID] = true
		}
	}
	if len(completedIDs) == 0 {
		return nil
	}
	originalProfiles := append([]Profile(nil), a.config.Profiles...)
	for _, profile := range originalProfiles {
		if !completedIDs[profile.ID] {
			continue
		}
		state := stateByID[profile.ID]
		archived := ArchivedProfile{
			ArchiveID:      fmt.Sprintf("archive_%s_%d", profile.ID, a.now().Unix()),
			FinishedAt:     a.now().Format(time.RFC3339),
			Reason:         "Zakonczony automatycznie po wykonaniu planu",
			Profile:        profile,
			OverdueCount:   state.OverdueCount,
			ExtensionDays:  state.ExtensionDays,
			CompletedCount: a.completedCountForProfile(profile),
		}
		archivedProgress := progressForProfile(a.progress, profile)
		if err := saveArchiveFolder(a.archivePath, archived, archivedProgress); err != nil {
			return err
		}
		archived.Progress = &archivedProgress
		if err := removeActiveProgressFile(a.progressPath, profile); err != nil {
			return err
		}
		a.archive.Profiles = append([]ArchivedProfile{archived}, a.archive.Profiles...)
	}
	remaining := make([]Profile, 0, len(originalProfiles)-len(completedIDs))
	for _, profile := range originalProfiles {
		if !completedIDs[profile.ID] {
			remaining = append(remaining, profile)
		}
	}
	a.config.Profiles = remaining
	for _, profile := range originalProfiles {
		if completedIDs[profile.ID] {
			a.progress = progressWithoutProfile(a.progress, profile)
		}
	}
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		a.config.Profiles = originalProfiles
		return err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return err
	}
	return nil
}

func profileScheduleEnded(profile Profile, startDate, today time.Time) bool {
	totalDays := 0
	for _, phase := range profile.Phases {
		if phase.DurationDays > 0 {
			totalDays += phase.DurationDays
		}
	}
	if totalDays == 0 {
		return false
	}
	lastPlannedDay := dateOnly(startDate).AddDate(0, 0, totalDays-1)
	return !dateOnly(today).Before(lastPlannedDay)
}

func (a *Application) completedCountForProfile(profile Profile) int {
	profileProgress := progressForProfile(a.progress, profile)
	count := 0
	modernDates := map[string]bool{}
	for _, completion := range profileProgress.Completions {
		count++
		if completedAt, valid := parseTimestamp(completion.CompletedAt); valid {
			modernDates[dateOnly(completedAt.In(a.now().Location())).Format("2006-01-02")] = true
		}
	}
	for date, entries := range profileProgress.History {
		if entry, exists := entries[profile.Name]; exists && entry.Done && !modernDates[date] {
			count++
		}
	}
	return count
}

func progressForProfile(source Progress, profile Profile) Progress {
	result := Progress{
		StartDate:         valueOr(profile.StartDate, source.StartDate),
		TrackingSince:     source.TrackingSince,
		History:           map[string]map[string]HistoryEntry{},
		Completions:       map[string]SessionCompletion{},
		PausedSessions:    map[string]PausedSession{},
		DismissedSessions: map[string]DismissedSession{},
	}
	for sessionID, paused := range source.PausedSessions {
		if paused.ProfileID == profile.ID && (profile.RunID == "" || paused.RunID == profile.RunID) {
			result.PausedSessions[sessionID] = paused
		}
	}
	for sessionID, dismissed := range source.DismissedSessions {
		if dismissed.ProfileID == profile.ID && (profile.RunID == "" || dismissed.RunID == profile.RunID) {
			result.DismissedSessions[sessionID] = dismissed
		}
	}
	for sessionID, completion := range source.Completions {
		if completion.ProfileID != profile.ID {
			continue
		}
		if profile.RunID != "" && completion.RunID != profile.RunID {
			continue
		}
		result.Completions[sessionID] = completion
	}
	for date, entries := range source.History {
		if entry, exists := entries[profile.Name]; exists {
			result.History[date] = map[string]HistoryEntry{profile.Name: entry}
		}
	}
	return result
}

func progressWithoutProfile(source Progress, profile Profile) Progress {
	result := Progress{
		StartDate:         source.StartDate,
		TrackingSince:     source.TrackingSince,
		History:           map[string]map[string]HistoryEntry{},
		Completions:       map[string]SessionCompletion{},
		PausedSessions:    map[string]PausedSession{},
		DismissedSessions: map[string]DismissedSession{},
	}
	for sessionID, dismissed := range source.DismissedSessions {
		matchesRun := profile.RunID == "" || dismissed.RunID == profile.RunID
		if dismissed.ProfileID == profile.ID && matchesRun {
			continue
		}
		result.DismissedSessions[sessionID] = dismissed
	}
	for sessionID, paused := range source.PausedSessions {
		matchesRun := profile.RunID == "" || paused.RunID == profile.RunID
		if paused.ProfileID == profile.ID && matchesRun {
			continue
		}
		result.PausedSessions[sessionID] = paused
	}
	for sessionID, completion := range source.Completions {
		matchesRun := profile.RunID == "" || completion.RunID == profile.RunID
		if completion.ProfileID == profile.ID && matchesRun {
			continue
		}
		result.Completions[sessionID] = completion
	}
	for date, entries := range source.History {
		kept := map[string]HistoryEntry{}
		for name, entry := range entries {
			if name != profile.Name {
				kept[name] = entry
			}
		}
		if len(kept) > 0 {
			result.History[date] = kept
		}
	}
	return result
}

// progressWithoutPerson czyści z postępu wszystko, co należy do usuwanej osoby
// (completions, pauzy, odpuszczenia). Historia wpisana po nazwie profilu nie ma
// pola PersonID, więc zostaje bez zmian — osoba bez aktywnego programu i tak nie
// ma już wpisów w bieżącym postępie.
func progressWithoutPerson(source Progress, personID string) Progress {
	result := Progress{
		StartDate:         source.StartDate,
		TrackingSince:     source.TrackingSince,
		History:           map[string]map[string]HistoryEntry{},
		Completions:       map[string]SessionCompletion{},
		PausedSessions:    map[string]PausedSession{},
		DismissedSessions: map[string]DismissedSession{},
	}
	for sessionID, dismissed := range source.DismissedSessions {
		if dismissed.PersonID == personID {
			continue
		}
		result.DismissedSessions[sessionID] = dismissed
	}
	for sessionID, paused := range source.PausedSessions {
		if paused.PersonID == personID {
			continue
		}
		result.PausedSessions[sessionID] = paused
	}
	for sessionID, completion := range source.Completions {
		if completion.PersonID == personID {
			continue
		}
		result.Completions[sessionID] = completion
	}
	result.History = source.History
	return result
}

func (a *Application) SetStartDate(value string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return Snapshot{}, errors.New("data startu ma nieprawidłowy format")
	}
	if err := a.archiveCurrentRunsLocked("Zakończony przy rozpoczęciu nowego planu"); err != nil {
		return Snapshot{}, err
	}
	if err := a.renewProfileRunsLocked(); err != nil {
		return Snapshot{}, err
	}
	a.progress.StartDate = parsed.Format("2006-01-02")
	a.progress.TrackingSince = parsed.Format("2006-01-02")
	for index := range a.config.Profiles {
		a.config.Profiles[index].StartDate = parsed.Format("2006-01-02")
	}
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) ResetProgress() (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.archiveCurrentRunsLocked("Zakończony przy wyzerowaniu postępu"); err != nil {
		return Snapshot{}, err
	}
	if err := a.renewProfileRunsLocked(); err != nil {
		return Snapshot{}, err
	}
	a.progress = defaultProgress(a.now())
	if err := saveJSONAtomic(a.profilesPath, a.config); err != nil {
		return Snapshot{}, err
	}
	if err := saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(30), nil
}

func (a *Application) renewProfileRunsLocked() error {
	for index := range a.config.Profiles {
		if err := removeActiveProgressFile(a.progressPath, a.config.Profiles[index]); err != nil {
			return err
		}
		a.config.Profiles[index].RunID = newRunID()
	}
	return nil
}

func (a *Application) archiveCurrentRunsLocked(reason string) error {
	for _, profile := range a.config.Profiles {
		profileProgress := progressForProfile(a.progress, profile)
		if len(profileProgress.Completions) == 0 && len(profileProgress.History) == 0 {
			continue
		}
		archived := ArchivedProfile{
			ArchiveID:      fmt.Sprintf("archive_%s_%d", profile.ID, a.now().UnixNano()),
			FinishedAt:     a.now().Format(time.RFC3339),
			Reason:         reason,
			Profile:        profile,
			CompletedCount: a.completedCountForProfile(profile),
		}
		if err := saveArchiveFolder(a.archivePath, archived, profileProgress); err != nil {
			return err
		}
		archived.Progress = &profileProgress
		if err := removeActiveProgressFile(a.progressPath, profile); err != nil {
			return err
		}
		a.archive.Profiles = append([]ArchivedProfile{archived}, a.archive.Profiles...)
		a.progress = progressWithoutProfile(a.progress, profile)
	}
	return nil
}

func (a *Application) snapshotLocked(days int) Snapshot {
	currentTime := a.now()
	now := dateOnly(currentTime)
	defaultStartDate := effectiveStartDate(a.config, a.progress, now)
	todayPlans, profileStates := buildOperationalPlans(a.config, a.progress, currentTime)
	var todayRemainingSeconds uint32
	for _, plan := range todayPlans {
		if plan.Status != "session" || plan.Done {
			continue
		}
		for _, step := range plan.DeviceSteps {
			todayRemainingSeconds += step.DurationSeconds
		}
	}
	schedules := make([]ProfileSchedule, 0, len(a.config.Profiles))

	for _, profile := range a.config.Profiles {
		startDate := profileStartDate(profile, defaultStartDate)
		profileSchedule := ProfileSchedule{
			ProfileName: profile.Name,
			Days:        make([]DayPlan, 0, days),
		}
		for offset := 0; offset < days; offset++ {
			current := now.AddDate(0, 0, offset)
			plan := planForDate(profile, current, startDate)
			profileSchedule.Days = append(profileSchedule.Days, applyScheduleProgress(plan, &a.progress))
		}
		schedules = append(schedules, profileSchedule)
	}

	return Snapshot{
		Config:                a.config,
		Progress:              a.progress,
		Today:                 todayPlans,
		Schedule:              schedules,
		Frequencies:           a.frequencies,
		ProfileStates:         profileStates,
		Archive:               append([]ArchivedProfile(nil), a.archive.Profiles...),
		Persons:               append([]Person(nil), a.persons.Persons...),
		ManualRuns:            append([]ManualRun(nil), a.manualRuns.Runs...),
		TodayRemainingSeconds: todayRemainingSeconds,
		Meta: AppMeta{
			DataDirectory: filepath.Dir(a.profilesPath),
			ProfilesFile:  a.profilesPath,
			ProgressFile:  a.progressPath,
			Version:       appVersion,
		},
	}
}
