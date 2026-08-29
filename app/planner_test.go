package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func plannerTestConfig(compatible bool) Config {
	compatibleWith := []string{}
	compatibleFromA := []string{}
	if compatible {
		compatibleWith = []string{"program_a"}
		compatibleFromA = []string{"program_b"}
	}
	config := Config{
		StartDate: "2026-08-03",
		Profiles: []Profile{{
			ID:   "profil_test",
			Name: "Test",
			Phases: []Phase{{
				ID:           "faza_test",
				Name:         "Faza",
				DurationDays: 7,
				Mode:         "weekly",
				Schedule: map[string]DailyPlan{
					"Monday": {
						Frequency:  "Program A",
						Time:       "5 min",
						Scheduling: SessionScheduling{ProgramID: "program_a", SameDayWith: compatibleFromA},
					},
					"Tuesday": {
						Frequency:  "Program B",
						Time:       "5 min",
						Scheduling: SessionScheduling{ProgramID: "program_b", SameDayWith: compatibleWith},
					},
				},
			}},
		}},
	}
	return config
}

func TestMissedSessionStaysInOperationalQueue(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	plans, states := buildOperationalPlans(plannerTestConfig(false), Progress{
		StartDate:     "2026-08-03",
		TrackingSince: "2026-08-03",
		Completions:   map[string]SessionCompletion{},
	}, now)
	if len(plans) != 2 {
		t.Fatalf("oczekiwano zaległej i dzisiejszej sesji, otrzymano %d: %#v", len(plans), plans)
	}
	if !plans[0].Overdue || plans[0].PlannedDate != "2026-08-03" || !plans[0].Available {
		t.Fatalf("pierwsza sesja powinna być dostępną zaległością: %#v", plans[0])
	}
	if states[0].ExtensionDays != 1 || states[0].OverdueCount != 1 {
		t.Fatalf("nieprawidłowy stan przedłużenia: %#v", states[0])
	}
}

func TestIncompatibleProgramWaitsAfterCompletion(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	config := plannerTestConfig(false)
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	progress := Progress{
		StartDate:     "2026-08-03",
		TrackingSince: "2026-08-03",
		Completions: map[string]SessionCompletion{
			mondayPlan.SessionID: {
				SessionID:   mondayPlan.SessionID,
				ProfileID:   "profil_test",
				ProgramID:   "program_a",
				CompletedAt: time.Date(2026, time.August, 4, 9, 0, 0, 0, time.Local).Format(time.RFC3339),
				Scheduling:  mondayPlan.Scheduling,
			},
		},
	}
	plans, _ := buildOperationalPlans(config, progress, now)
	var tuesday DayPlan
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" {
			tuesday = plan
		}
	}
	if tuesday.Available || tuesday.BlockedReason == "" {
		t.Fatalf("niezgodny program powinien czekać: %#v", tuesday)
	}
}

func TestCompatibleProgramCanRunSameDay(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	config := plannerTestConfig(true)
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	progress := Progress{
		StartDate:     "2026-08-03",
		TrackingSince: "2026-08-03",
		Completions: map[string]SessionCompletion{
			mondayPlan.SessionID: {
				SessionID:   mondayPlan.SessionID,
				ProfileID:   "profil_test",
				ProgramID:   "program_a",
				CompletedAt: time.Date(2026, time.August, 4, 9, 0, 0, 0, time.Local).Format(time.RFC3339),
				Scheduling:  mondayPlan.Scheduling,
			},
		},
	}
	plans, _ := buildOperationalPlans(config, progress, now)
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" && !plan.Available {
			t.Fatalf("zgodny program powinien być dostępny: %#v", plan)
		}
	}
}

func TestOneSidedSameDayDeclarationDoesNotUnlockPair(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	config := plannerTestConfig(false)
	tuesday := config.Profiles[0].Phases[0].Schedule["Tuesday"]
	tuesday.Scheduling.SameDayWith = []string{"program_a"}
	config.Profiles[0].Phases[0].Schedule["Tuesday"] = tuesday
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-03", Completions: map[string]SessionCompletion{
		mondayPlan.SessionID: {
			SessionID: mondayPlan.SessionID, SessionGroupID: mondayPlan.SessionGroupID, ProfileID: "profil_test", ProgramID: "program_a",
			CompletedAt: time.Date(2026, time.August, 4, 9, 0, 0, 0, time.Local).Format(time.RFC3339), Scheduling: mondayPlan.Scheduling,
		},
	}}
	plans, _ := buildOperationalPlans(config, progress, now)
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" && plan.Available {
			t.Fatalf("jednostronna zgodność nie powinna odblokować pary: %#v", plan)
		}
	}
}

func TestCooldownBlocksDifferentBacklogProgramAcrossDays(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	config := plannerTestConfig(false)
	monday := config.Profiles[0].Phases[0].Schedule["Monday"]
	monday.Scheduling.CooldownAfterMinutes = 2880
	config.Profiles[0].Phases[0].Schedule["Monday"] = monday
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-03", Completions: map[string]SessionCompletion{
		mondayPlan.SessionID: {
			SessionID: mondayPlan.SessionID, SessionGroupID: mondayPlan.SessionGroupID, ProfileID: "profil_test", ProgramID: "program_a",
			CompletedAt: time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local).Format(time.RFC3339), Scheduling: mondayPlan.Scheduling,
		},
	}}
	plans, _ := buildOperationalPlans(config, progress, now)
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" {
			if plan.Available || plan.AvailableAt == "" {
				t.Fatalf("cooldown powinien zablokować kolejny program: %#v", plan)
			}
			return
		}
	}
	t.Fatal("nie znaleziono wtorkowej sesji")
}

func TestMinGapStillAppliesBetweenSeparateSessionGroups(t *testing.T) {
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	shared := SessionScheduling{ProgramID: "program_a", MinGapMinutes: 2880}
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Faza", DurationDays: 7, Mode: "weekly",
			Schedule: map[string]DailyPlan{
				"Monday":  {Frequency: "Program A", Time: "5 min", Scheduling: shared},
				"Tuesday": {Frequency: "Program A — ponownie", Time: "5 min", Scheduling: shared},
			},
		}},
	}}}
	mondayPlan := expandScheduledPlan(planForDate(config.Profiles[0], start, start))[0]
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-03", Completions: map[string]SessionCompletion{
		mondayPlan.SessionID: {
			SessionID: mondayPlan.SessionID, SessionGroupID: mondayPlan.SessionGroupID, ProfileID: "profil_test", ProgramID: "program_a",
			CompletedAt: time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local).Format(time.RFC3339), Scheduling: shared,
		},
	}}
	plans, _ := buildOperationalPlans(config, progress, now)
	for _, plan := range plans {
		if plan.PlannedDate == "2026-08-04" {
			if plan.Available || plan.AvailableAt == "" {
				t.Fatalf("odstęp między pełnymi sesjami powinien nadal obowiązywać: %#v", plan)
			}
			return
		}
	}
	t.Fatal("nie znaleziono drugiej sesji")
}

func TestRepeatedSessionEnforcesOrderAndBreak(t *testing.T) {
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Faza", DurationDays: 1, Mode: "interval", Program: "Program A", Time: "7 min",
			Scheduling: SessionScheduling{ProgramID: "program_a", Repetitions: 3, BreakBetweenMinutes: 20, MinGapMinutes: 2880},
		}},
	}}}
	progress := Progress{StartDate: "2026-08-03", TrackingSince: "2026-08-03", Completions: map[string]SessionCompletion{}}
	plans, _ := buildOperationalPlans(config, progress, time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local))
	if len(plans) != 3 || !plans[0].Available || plans[1].Available || plans[2].Available {
		t.Fatalf("na początku dostępna powinna być tylko pierwsza część: %#v", plans)
	}
	progress.Completions[plans[0].SessionID] = SessionCompletion{
		SessionID: plans[0].SessionID, ProfileID: "profil_test", ProgramID: "program_a",
		CompletedAt: time.Date(2026, time.August, 3, 9, 0, 0, 0, time.Local).Format(time.RFC3339), Scheduling: plans[0].Scheduling,
	}
	plans, _ = buildOperationalPlans(config, progress, time.Date(2026, time.August, 3, 9, 10, 0, 0, time.Local))
	second := planWithRepeat(plans, 2)
	if second.Available || second.AvailableAt == "" {
		t.Fatalf("druga część powinna czekać na przerwę: %#v", second)
	}
	plans, _ = buildOperationalPlans(config, progress, time.Date(2026, time.August, 3, 9, 21, 0, 0, time.Local))
	second = planWithRepeat(plans, 2)
	if !second.Available {
		t.Fatalf("druga część powinna być dostępna po przerwie: %#v", second)
	}
}

func planWithRepeat(plans []DayPlan, repeat int) DayPlan {
	for _, plan := range plans {
		if plan.RepeatIndex == repeat {
			return plan
		}
	}
	return DayPlan{}
}

func TestFinishProfileMovesItToLocalArchive(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time { return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local) }
	config := plannerTestConfig(false)
	snapshot, err := application.SaveConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	profileID := snapshot.Config.Profiles[0].ID
	snapshot, err = application.FinishProfile(profileID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Config.Profiles) != 0 || len(snapshot.Archive) != 1 {
		t.Fatalf("profil nie został przeniesiony do archiwum: %#v", snapshot)
	}
	archiveID := snapshot.Archive[0].ArchiveID
	var archived ArchivedProfile
	if err := loadJSON(filepath.Join(directory, "data", "archive", archiveID, "profile.json"), &archived); err != nil {
		t.Fatal(err)
	}
	if archived.Profile.ID != profileID {
		t.Fatalf("nieprawidłowy zapis archiwum: %#v", archived)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "archive", archiveID, "progress.json")); err != nil {
		t.Fatalf("brak progresu obok archiwalnego profilu: %v", err)
	}
}

func TestNewProfileRunDoesNotInheritCompletion(t *testing.T) {
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)
	config := plannerTestConfig(false)
	config.Profiles[0].RunID = "run_old"
	oldPlan := expandScheduledPlan(planForDate(config.Profiles[0], start, start))[0]
	progress := Progress{
		StartDate:     "2026-08-03",
		TrackingSince: "2026-08-03",
		Completions: map[string]SessionCompletion{
			oldPlan.SessionID: {
				SessionID: oldPlan.SessionID,
				ProfileID: oldPlan.ProfileID,
				RunID:     oldPlan.RunID,
			},
		},
	}

	config.Profiles[0].RunID = "run_new"
	plans, _ := buildOperationalPlans(config, progress, time.Date(2026, time.August, 3, 12, 0, 0, 0, time.Local))
	if len(plans) == 0 || plans[0].Done {
		t.Fatalf("nowy przebieg nie może odziedziczyć wykonania starego: %#v", plans)
	}
	if plans[0].SessionID == oldPlan.SessionID {
		t.Fatalf("nowy przebieg powinien mieć inne ID sesji: %q", plans[0].SessionID)
	}
}

func TestCompletedProfileArchivesAutomaticallyAfterLastSession(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.Local)
	application.now = func() time.Time { return current }
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_auto", Name: "Auto", Phases: []Phase{{
			ID: "faza_auto", Name: "Faza", DurationDays: 2, Mode: "interval", Program: "Program A", Time: "5 min",
			Scheduling: SessionScheduling{ProgramID: "program_a"},
		}},
	}}}
	if _, err := application.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	application.progress.StartDate = "2026-08-03"
	application.progress.TrackingSince = "2026-08-03"
	monday := expandScheduledPlan(planForDate(application.config.Profiles[0], time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	tuesday := expandScheduledPlan(planForDate(application.config.Profiles[0], time.Date(2026, time.August, 4, 0, 0, 0, 0, time.Local), time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)))[0]
	if _, err := application.SetSessionDone(monday.SessionID, true); err != nil {
		t.Fatal(err)
	}
	current = time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	if _, err := application.SetSessionDone(tuesday.SessionID, true); err != nil {
		t.Fatal(err)
	}
	snapshot := application.Snapshot()
	if len(snapshot.Config.Profiles) != 0 || len(snapshot.Archive) != 1 || snapshot.Archive[0].Reason == "Zakonczony rÄ™cznie" {
		t.Fatalf("profil powinien trafić do archiwum automatycznie: %#v", snapshot)
	}
}

func TestApplyScheduleProgressMarksDoneDay(t *testing.T) {
	profile := Profile{
		ID: "p", Name: "Maks", PersonID: "maks", RunID: "run1", StartDate: "2026-08-01",
		Phases: []Phase{{ID: "ph", Name: "Start", Mode: "interval", DurationDays: 10, IntervalGap: 0, Program: "30 kHz", Time: "7 min", Scheduling: SessionScheduling{Repetitions: 2}, DeviceSteps: []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 420}}}},
	}
	startDate := parseDate("2026-08-01", time.Now())
	day := planForDate(profile, parseDate("2026-08-03", time.Now()), startDate)

	progress := &Progress{Completions: map[string]SessionCompletion{
		sessionIDFor(day, 1): {RunID: "run1"},
		sessionIDFor(day, 2): {RunID: "run1"},
	}}
	result := applyScheduleProgress(day, progress)
	if !result.Done {
		t.Fatalf("dzień z oboma powtórzeniami wykonanymi powinien być done: %#v", result)
	}

	progress.Completions = map[string]SessionCompletion{
		sessionIDFor(day, 1): {RunID: "run1"},
	}
	result = applyScheduleProgress(day, progress)
	if result.Done || result.Note != "Wykonano 1 z 2" {
		t.Fatalf("częściowe wykonanie powinno zostać niedone z notatką: %#v", result)
	}

	progress.Completions = map[string]SessionCompletion{
		sessionIDFor(day, 1): {RunID: "stary_przebieg"},
		sessionIDFor(day, 2): {RunID: "stary_przebieg"},
	}
	result = applyScheduleProgress(day, progress)
	if result.Done {
		t.Fatalf("wykonania z innego przebiegu nie mogą oznaczyć dnia: %#v", result)
	}
}
