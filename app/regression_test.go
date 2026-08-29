package main

import (
	"strings"
	"testing"
	"time"
)

func warsaw(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Skipf("brak bazy stref czasowych: %v", err)
	}
	return location
}

// === 1. Zmiana czasu nie może przesuwać planu o dobę ===

// Doba przejścia na czas letni ma 23 h, a powrotu na zimowy 25 h. Liczenie
// numeru dnia przez Sub().Hours()/24 gubiło albo dublowało całą dobę: numer dnia
// powtarzał się, sesja przeskakiwała o dzień, a granice faz się przesuwały.
func TestDayNumbersStayMonotonicAcrossDaylightSavingChanges(t *testing.T) {
	location := warsaw(t)
	cases := []struct {
		name  string
		start time.Time
		days  int
	}{
		{"przejście na czas letni 2026-03-29", time.Date(2026, time.March, 20, 0, 0, 0, 0, location), 20},
		{"powrót na czas zimowy 2026-10-25", time.Date(2026, time.October, 18, 0, 0, 0, 0, location), 20},
	}
	profile := Profile{
		ID:   "profil_dst",
		Name: "DST",
		Phases: []Phase{
			{ID: "faza_1", Name: "Faza 1", DurationDays: 10, Mode: "interval", IntervalGap: 1, Program: "Program A"},
			{ID: "faza_2", Name: "Faza 2", DurationDays: 10, Mode: "interval", IntervalGap: 1, Program: "Program B"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			previousDayNumber := 0
			for offset := 0; offset < testCase.days; offset++ {
				current := testCase.start.AddDate(0, 0, offset)
				plan := planForDate(profile, current, testCase.start)
				expected := offset + 1
				if plan.DayNumber != expected {
					t.Fatalf("%s: numer dnia %d, oczekiwano %d", current.Format("2006-01-02"), plan.DayNumber, expected)
				}
				if offset > 0 && plan.DayNumber != previousDayNumber+1 {
					t.Fatalf("%s: numer dnia nie wzrósł o 1 (%d po %d)", current.Format("2006-01-02"), plan.DayNumber, previousDayNumber)
				}
				previousDayNumber = plan.DayNumber
				// Rytm interwału: sesja co drugi dzień, licząc od dnia 1.
				wantSession := offset%2 == 0
				if wantSession && plan.Status != "session" {
					t.Fatalf("%s (dzień %d) powinien być dniem sesyjnym, jest %q", current.Format("2006-01-02"), plan.DayNumber, plan.Status)
				}
				if !wantSession && plan.Status != "break" {
					t.Fatalf("%s (dzień %d) powinien być dniem przerwy, jest %q", current.Format("2006-01-02"), plan.DayNumber, plan.Status)
				}
				// Granica faz: dni 1–10 to faza 1, dni 11–20 to faza 2.
				wantPhase := "Faza 1"
				if plan.DayNumber > 10 {
					wantPhase = "Faza 2"
				}
				if plan.PhaseName != wantPhase {
					t.Fatalf("%s (dzień %d) trafił do fazy %q, oczekiwano %q", current.Format("2006-01-02"), plan.DayNumber, plan.PhaseName, wantPhase)
				}
			}
		})
	}
}

// Błąd nie znikał po jednym dniu: obcięcie doby 23-godzinnej przesuwało numerację
// na CAŁY okres czasu letniego i kompensowało się dopiero przy powrocie na zimowy.
// Ten test pilnuje całego przedziału marzec–listopad, po obu stronach obu zmian.
func TestDayNumberIsCorrectThroughEntireSummerTimePeriod(t *testing.T) {
	location := warsaw(t)
	start := time.Date(2026, time.March, 1, 0, 0, 0, 0, location)
	profile := Profile{ID: "profil_dst", Name: "DST", Phases: []Phase{
		{ID: "faza_1", Name: "Faza 1", DurationDays: 300, Mode: "interval", IntervalGap: 1, Program: "Program A"},
	}}
	for offset := 0; offset <= 260; offset++ {
		current := start.AddDate(0, 0, offset)
		plan := planForDate(profile, current, start)
		if plan.DayNumber != offset+1 {
			t.Fatalf("%s: numer dnia %d, oczekiwano %d", current.Format("2006-01-02"), plan.DayNumber, offset+1)
		}
		wantStatus := "break"
		if offset%2 == 0 {
			wantStatus = "session"
		}
		if plan.Status != wantStatus {
			t.Fatalf("%s (dzień %d): status %q, oczekiwano %q — parzystość dni sesyjnych się odwróciła",
				current.Format("2006-01-02"), plan.DayNumber, plan.Status, wantStatus)
		}
	}
}

func TestDaysBetweenIgnoresDaylightSavingShift(t *testing.T) {
	location := warsaw(t)
	summerShift := daysBetween(
		time.Date(2026, time.March, 29, 0, 0, 0, 0, location),
		time.Date(2026, time.March, 30, 0, 0, 0, 0, location))
	if summerShift != 1 {
		t.Fatalf("doba 23-godzinna policzona jako %d dni", summerShift)
	}
	winterShift := daysBetween(
		time.Date(2026, time.October, 25, 0, 0, 0, 0, location),
		time.Date(2026, time.October, 26, 0, 0, 0, 0, location))
	if winterShift != 1 {
		t.Fatalf("doba 25-godzinna policzona jako %d dni", winterShift)
	}
}

// === 2. same_day_with musi się normalizować także w trybie tygodniowym ===

func weeklySameDayConfig(mondayPartners, tuesdayPartners []string) Config {
	return Config{
		StartDate: "2026-08-03",
		Profiles: []Profile{{
			ID: "profil_test", Name: "Test", Phases: []Phase{{
				ID: "faza_test", Name: "Faza", DurationDays: 7, Mode: "weekly",
				Schedule: map[string]DailyPlan{
					"Monday":  {Frequency: "Program A", Time: "5 min", Scheduling: SessionScheduling{SameDayWith: mondayPartners}},
					"Tuesday": {Frequency: "Program B", Time: "5 min", Scheduling: SessionScheduling{SameDayWith: tuesdayPartners}},
				},
			}},
		}},
	}
}

// `daily` w pętli range po mapie to KOPIA. Normalizacja SameDayWith zapisywała się
// tylko na kopii, więc runtime porównywał "program_b" z wpisanym "Program B"
// i para "można tego samego dnia" nigdy nie działała.
func TestWeeklyScheduleNormalizesSameDayWithInMap(t *testing.T) {
	config := weeklySameDayConfig([]string{"Program B"}, []string{"Program A"})
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("poprawna konfiguracja została odrzucona: %v", err)
	}
	monday := config.Profiles[0].Phases[0].Schedule["Monday"]
	tuesday := config.Profiles[0].Phases[0].Schedule["Tuesday"]
	if len(monday.Scheduling.SameDayWith) != 1 || monday.Scheduling.SameDayWith[0] != "program_b" {
		t.Fatalf("mapa nie zawiera znormalizowanej wartości: %#v", monday.Scheduling.SameDayWith)
	}
	if len(tuesday.Scheduling.SameDayWith) != 1 || tuesday.Scheduling.SameDayWith[0] != "program_a" {
		t.Fatalf("mapa nie zawiera znormalizowanej wartości: %#v", tuesday.Scheduling.SameDayWith)
	}
	if !programsCompatible(monday.Scheduling, tuesday.Scheduling) {
		t.Fatalf("runtime nie rozpoznaje pary jako zgodnej: %#v vs %#v", monday.Scheduling, tuesday.Scheduling)
	}
}

func TestWeeklyScheduleDropsSelfReferenceAndDuplicates(t *testing.T) {
	config := weeklySameDayConfig([]string{"Program A", "Program B", "program_b"}, []string{"Program A"})
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("konfiguracja została odrzucona: %v", err)
	}
	monday := config.Profiles[0].Phases[0].Schedule["Monday"]
	if len(monday.Scheduling.SameDayWith) != 1 || monday.Scheduling.SameDayWith[0] != "program_b" {
		t.Fatalf("self-referencja lub duplikat zostały zapisane: %#v", monday.Scheduling.SameDayWith)
	}
}

// === 3. Zgodność same_day_with nie znosi jawnego cooldownu ===

func compatiblePairWithCooldown(t *testing.T, cooldownMinutes int, now time.Time) DayPlan {
	t.Helper()
	config := weeklySameDayConfig([]string{"Program B"}, []string{"Program A"})
	monday := config.Profiles[0].Phases[0].Schedule["Monday"]
	monday.Scheduling.CooldownAfterMinutes = cooldownMinutes
	config.Profiles[0].Phases[0].Schedule["Monday"] = monday
	tuesday := config.Profiles[0].Phases[0].Schedule["Tuesday"]
	tuesday.Scheduling.CooldownAfterMinutes = cooldownMinutes
	config.Profiles[0].Phases[0].Schedule["Tuesday"] = tuesday
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("konfiguracja została odrzucona: %v", err)
	}
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], start, start))[0]
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-04", Completions: map[string]SessionCompletion{
		mondayPlan.SessionID: {
			SessionID: mondayPlan.SessionID, SessionGroupID: mondayPlan.SessionGroupID,
			ProfileID: "profil_test", RunID: mondayPlan.RunID, ProgramID: "program_a",
			CompletedAt: time.Date(2026, time.August, 4, 8, 0, 0, 0, time.Local).Format(time.RFC3339),
			Scheduling:  mondayPlan.Scheduling,
		},
	}}
	plans, _ := buildOperationalPlans(config, progress, now)
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" {
			return plan
		}
	}
	t.Fatal("nie znaleziono wtorkowej sesji")
	return DayPlan{}
}

// Sprawdzenie zgodności stało przed naliczeniem cooldownu, więc para A↔B
// z cooldownem 240 min była dostępna 5 minut po A. Użytkownik ustawiał 4 h i dostawał 0.
func TestCompatiblePairStillRespectsCooldown(t *testing.T) {
	// 5 minut po sesji A — cooldown 240 min ma jeszcze obowiązywać.
	tooEarly := compatiblePairWithCooldown(t, 240, time.Date(2026, time.August, 4, 8, 5, 0, 0, time.Local))
	if tooEarly.Available {
		t.Fatalf("cooldown 240 min został zignorowany dla pary zgodnej: %#v", tooEarly)
	}
	if !strings.Contains(tooEarly.BlockedReason, "240") {
		t.Fatalf("powód blokady nie wskazuje cooldownu: %q", tooEarly.BlockedReason)
	}
	// Po upływie 240 minut sesja ma być dostępna.
	afterCooldown := compatiblePairWithCooldown(t, 240, time.Date(2026, time.August, 4, 12, 1, 0, 0, time.Local))
	if !afterCooldown.Available {
		t.Fatalf("po upływie cooldownu sesja powinna być dostępna: %#v", afterCooldown)
	}
}

// Zachowanie bez cooldownu pozostaje bez zmian: para zgodna działa tego samego dnia.
func TestCompatiblePairWithoutCooldownIsImmediatelyAvailable(t *testing.T) {
	plan := compatiblePairWithCooldown(t, 0, time.Date(2026, time.August, 4, 8, 5, 0, 0, time.Local))
	if !plan.Available {
		t.Fatalf("para zgodna bez cooldownu powinna być dostępna od razu: %#v", plan)
	}
}

// === 5. Walidacja krzyżowa: konfiguracje wewnętrznie sprzeczne ===

func intervalPhaseConfig(intervalGap, durationDays, minGapMinutes int) Config {
	return Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Faza intensywna", DurationDays: durationDays, Mode: "interval",
			IntervalGap: intervalGap, Program: "Program A", Time: "5 min",
			Scheduling: SessionScheduling{ProgramID: "program_a", MinGapMinutes: minGapMinutes},
		}},
	}}}
}

// Plan generował terminy, których minimalny odstęp nigdy nie pozwalał dotrzymać —
// każda kolejna sesja od razu wpadała w zaległości, bez żadnego ostrzeżenia.
func TestContradictoryMinGapIsRejected(t *testing.T) {
	// Sesje co 2 dni (2880 min), a minimalny odstęp 4320 min — nie do wykonania.
	config := intervalPhaseConfig(1, 14, 4320)
	_, err := validateConfig(&config, validationModeSave)
	if err == nil {
		t.Fatal("sprzeczna konfiguracja została przyjęta")
	}
	for _, fragment := range []string{"Faza intensywna", "minimalny odstęp", "4320"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("komunikat %q nie zawiera %q", err.Error(), fragment)
		}
	}
}

func TestMinGapEqualToIntervalPeriodIsAccepted(t *testing.T) {
	// Przypadek graniczny: odstęp dokładnie równy okresowi między sesjami.
	config := intervalPhaseConfig(1, 14, 2880)
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("graniczna konfiguracja została odrzucona: %v", err)
	}
	// Wariant z zadania: min_gap == interval_gap * 1440.
	config = intervalPhaseConfig(1, 14, 1440)
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("konfiguracja min_gap = interval_gap*1440 została odrzucona: %v", err)
	}
	// Faza mieszcząca tylko jeden termin nie może być sprzeczna.
	config = intervalPhaseConfig(0, 1, 2880)
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("faza z jednym terminem została odrzucona: %v", err)
	}
}

func TestRepetitionsThatDoNotFitBeforeNextSessionAreRejected(t *testing.T) {
	config := intervalPhaseConfig(0, 7, 0)
	config.Profiles[0].Phases[0].Scheduling.Repetitions = 4
	config.Profiles[0].Phases[0].Scheduling.BreakBetweenMinutes = 600
	_, err := validateConfig(&config, validationModeSave)
	if err == nil {
		t.Fatal("powtórzenia niemieszczące się przed kolejnym terminem zostały przyjęte")
	}
	if !strings.Contains(err.Error(), "Faza intensywna") {
		t.Fatalf("komunikat nie wskazuje fazy: %q", err.Error())
	}
}

// === 6. Zaległości muszą dać się domknąć ===

func overdueApplication(t *testing.T) (*Application, string, time.Time) {
	t.Helper()
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	application.now = func() time.Time { return current }
	config := Config{StartDate: "2026-07-20", Profiles: []Profile{{
		ID: "profil_zalegly", Name: "Zaległy", StartDate: "2026-07-20", Phases: []Phase{{
			ID: "faza_zalegla", Name: "Faza", DurationDays: 5, Mode: "interval", Program: "Program A", Time: "5 min",
			Scheduling: SessionScheduling{ProgramID: "program_a"},
		}},
	}}}
	if _, err := application.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	application.progress.StartDate = "2026-07-20"
	application.progress.TrackingSince = "2026-07-20"
	return application, application.config.Profiles[0].ID, current
}

func TestDismissedOverdueSessionsLetProfileArchive(t *testing.T) {
	application, profileID, _ := overdueApplication(t)
	before := application.Snapshot()
	overdue := 0
	for _, plan := range before.Today {
		if plan.Status == "session" && !plan.Done && plan.Overdue {
			overdue++
		}
	}
	if overdue < 5 {
		t.Fatalf("scenariusz wymaga zaległości, jest ich %d", overdue)
	}
	if len(before.Config.Profiles) != 1 {
		t.Fatalf("profil nie powinien być jeszcze zarchiwizowany: %#v", before.Config.Profiles)
	}

	after, err := application.DismissOverdueSessions(profileID)
	if err != nil {
		t.Fatalf("nie udało się odpuścić zaległości: %v", err)
	}
	if len(after.Config.Profiles) != 0 {
		t.Fatalf("po odpuszczeniu zaległości profil powinien się zarchiwizować: %#v", after.Config.Profiles)
	}
	if len(after.Archive) != 1 {
		t.Fatalf("brak profilu w archiwum: %#v", after.Archive)
	}
	// Odpuszczone sesje NIE mogą być liczone jako wykonane.
	if after.Archive[0].CompletedCount != 0 {
		t.Fatalf("odpuszczone sesje policzono jako wykonane: %d", after.Archive[0].CompletedCount)
	}
	if archived := after.Archive[0].Progress; archived != nil {
		if len(archived.Completions) != 0 {
			t.Fatalf("odpuszczone sesje trafiły do wykonań: %#v", archived.Completions)
		}
		if len(archived.DismissedSessions) < 5 {
			t.Fatalf("odpuszczone sesje nie zostały zapisane osobno: %#v", archived.DismissedSessions)
		}
	}
}

func TestDismissedSessionsDisappearFromOperationalQueue(t *testing.T) {
	application, profileID, _ := overdueApplication(t)
	if _, err := application.DismissOverdueSessions(profileID); err != nil {
		t.Fatal(err)
	}
	for _, plan := range application.Snapshot().Today {
		if plan.ProfileID == profileID && plan.Status == "session" && !plan.Done {
			t.Fatalf("odpuszczona sesja wróciła do kolejki: %#v", plan)
		}
	}
}

func TestDismissOneRepeatedSessionDismissesWholeGroup(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local)
	application.now = func() time.Time { return current }
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "seria", Name: "Seria", StartDate: "2026-08-03", Phases: []Phase{{
			ID: "faza", Name: "Faza", DurationDays: 1, Mode: "interval", Program: "Clark", Time: "3x 7 min",
			Scheduling: SessionScheduling{ProgramID: "clark", Repetitions: 3, BreakBetweenMinutes: 20},
		}},
	}}}
	if _, err := application.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	snapshot := application.Snapshot()
	if len(snapshot.Today) != 3 {
		t.Fatalf("oczekiwano trzech technicznych części serii: %#v", snapshot.Today)
	}
	_, err = application.DismissSessionGroup(snapshot.Today[0].SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(application.progress.DismissedSessions) != 3 {
		t.Fatalf("seria nie została odpuszczona jako jeden termin: %#v", application.progress.DismissedSessions)
	}
	if len(application.progress.Completions) != 0 {
		t.Fatalf("odpuszczona seria nie może być wykonaniem: %#v", application.progress.Completions)
	}
}

func TestLaterDifferentProgramIsMarkedOutOfOrder(t *testing.T) {
	config := plannerTestConfig(false)
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-03", Completions: map[string]SessionCompletion{}}
	plans, _ := buildOperationalPlans(config, progress, time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local))
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" {
			if plan.OlderPendingCount != 1 {
				t.Fatalf("późniejsza sesja powinna wskazywać jeden starszy termin: %#v", plan)
			}
			return
		}
	}
	t.Fatal("nie znaleziono późniejszej sesji")
}

// === 7. Ręczne uruchomienie zostawia ślad ===

func TestManualRunIsRecordedSeparatelyFromPlannedSessions(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, time.August, 4, 15, 30, 0, 0, time.Local)
	application.now = func() time.Time { return current }
	if err := application.RecordManualRun(30_000, 420); err != nil {
		t.Fatal(err)
	}
	snapshot := application.Snapshot()
	if len(snapshot.ManualRuns) != 1 {
		t.Fatalf("ręczne uruchomienie nie zostawiło śladu: %#v", snapshot.ManualRuns)
	}
	run := snapshot.ManualRuns[0]
	if run.FrequencyMilliHz != 30_000 || run.DurationSeconds != 420 || run.Source != "manual" {
		t.Fatalf("nieprawidłowy zapis ręcznego uruchomienia: %#v", run)
	}
	if run.StartedAt != current.Format(time.RFC3339) {
		t.Fatalf("nie zapisano momentu uruchomienia: %#v", run)
	}
	// Ręczny przebieg nie może udawać wykonania zaplanowanej sesji.
	if len(snapshot.Progress.Completions) != 0 {
		t.Fatalf("ręczne uruchomienie trafiło do wykonań planu: %#v", snapshot.Progress.Completions)
	}
	// Ślad musi przetrwać restart aplikacji.
	reopened, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Snapshot().ManualRuns) != 1 {
		t.Fatal("ślad ręcznego uruchomienia nie przetrwał restartu")
	}
}

// === 8. Zaległości odblokowują się po jednej ===

func TestOnlyOldestOverdueSessionIsAvailable(t *testing.T) {
	application, profileID, _ := overdueApplication(t)
	plans := application.Snapshot().Today
	available := make([]DayPlan, 0)
	pending := 0
	for _, plan := range plans {
		if plan.ProfileID != profileID || plan.Status != "session" || plan.Done {
			continue
		}
		pending++
		if plan.Available {
			available = append(available, plan)
		}
	}
	if pending < 5 {
		t.Fatalf("scenariusz wymaga co najmniej 5 oczekujących sesji, jest %d", pending)
	}
	if len(available) != 1 {
		t.Fatalf("dostępna powinna być dokładnie jedna sesja, jest %d: %#v", len(available), available)
	}
	if available[0].PlannedDate != "2026-07-20" {
		t.Fatalf("dostępna powinna być najstarsza zaległość, jest z %s", available[0].PlannedDate)
	}
	for _, plan := range plans {
		if plan.ProfileID != profileID || plan.Status != "session" || plan.Done || plan.Available {
			continue
		}
		if !strings.Contains(plan.BlockedReason, "20.07.2026") {
			t.Fatalf("powód blokady nie wskazuje zaległości do wykonania: %q", plan.BlockedReason)
		}
	}
}

// === 9. Przebieg dzielony pauzą jest oznaczany w historii ===

func TestPausedAndResumedSessionIsMarkedAsSplit(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local)
	application.now = func() time.Time { return current }
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_pauza", Name: "Pauza", StartDate: "2026-08-03", Phases: []Phase{{
			ID: "faza_pauza", Name: "Faza", DurationDays: 3, Mode: "interval", Program: "Program A", Time: "10 min",
			DeviceSteps: []DeviceStep{{FrequencyMilliHz: 30_000, DurationSeconds: 600}},
			Scheduling:  SessionScheduling{ProgramID: "program_a"},
		}},
	}}}
	if _, err := application.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	application.progress.StartDate = "2026-08-03"
	application.progress.TrackingSince = "2026-08-03"
	sessionID := ""
	for _, plan := range application.Snapshot().Today {
		if plan.Status == "session" {
			sessionID = plan.SessionID
			break
		}
	}
	if sessionID == "" {
		t.Fatal("brak sesji do wstrzymania")
	}
	pausedAt := current
	if _, err := application.SavePausedSession(DevicePauseState{
		SessionID:        sessionID,
		RemainingSteps:   []DeviceStep{{FrequencyMilliHz: 30_000, DurationSeconds: 300}},
		RemainingSeconds: 300,
	}, false); err != nil {
		t.Fatal(err)
	}
	// Dokończenie po tygodniu nie może wyglądać jak jedno ciągłe wykonanie.
	current = time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	if _, err := application.SetSessionDone(sessionID, true); err != nil {
		t.Fatal(err)
	}
	completion, exists := application.progress.Completions[sessionID]
	if !exists {
		t.Fatal("brak zapisu wykonania")
	}
	if !completion.Split {
		t.Fatalf("przebieg dzielony pauzą nie został oznaczony: %#v", completion)
	}
	if completion.FirstStartedAt != pausedAt.Format(time.RFC3339) {
		t.Fatalf("nie zapisano momentu rozpoczęcia dzielonego przebiegu: %#v", completion)
	}
}

// === 10. Polityki programów zbierane zgodnie z trybem fazy ===

// planForDate w fazie interwałowej w ogóle nie czyta mapy Schedule, więc dane
// z tej mapy nie mogą wywracać walidacji konfiguracji, której aplikacja nie użyje.
func TestIntervalPhaseIgnoresLeftoverWeeklyScheduleInPolicies(t *testing.T) {
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Faza", DurationDays: 7, Mode: "interval", Program: "Program A", Time: "5 min",
			Scheduling: SessionScheduling{ProgramID: "program_a"},
			// Pozostałość po trybie tygodniowym: wskazuje program, którego w tej fazie nie ma.
			Schedule: map[string]DailyPlan{
				"Monday": {Frequency: "Program X", Scheduling: SessionScheduling{ProgramID: "program_x", SameDayWith: []string{"program_nieistniejacy"}}},
			},
		}},
	}}}
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("nieużywane dane tygodniowe wywróciły walidację fazy interwałowej: %v", err)
	}
}

// === 11. Optymalizacja evaluateAvailability nie zmienia zachowania ===

func TestCompletionIndexCoversEveryCompletion(t *testing.T) {
	completions := map[string]SessionCompletion{
		"a:1": {SessionID: "a:1", ProfileID: "p1"},
		"a:2": {SessionID: "a:2", ProfileID: "p1"},
		"b:1": {SessionID: "b:1", ProfileID: "p2"},
	}
	index := buildCompletionIndex(completions)
	if len(index.byProfile["p1"]) != 2 || len(index.byProfile["p2"]) != 1 {
		t.Fatalf("indeks zgubił wykonania: %#v", index.byProfile)
	}
}
