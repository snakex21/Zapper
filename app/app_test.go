package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplicationPersistsConfigurationAndProgress(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{
		StartDate: "2026-08-03",
		Profiles: []Profile{{
			Name: "Test",
			Phases: []Phase{{
				Name:         "Start",
				DurationDays: 7,
				Mode:         "interval",
				Program:      "30 kHz",
				Time:         "7 min",
				Schedule:     map[string]DailyPlan{},
			}},
		}},
	}
	if _, err := application.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "backups", "profiles.json.bak")); err != nil {
		t.Fatalf("brak lokalnej kopii zapasowej: %v", err)
	}

	application.now = func() time.Time {
		return time.Date(2026, time.August, 3, 12, 0, 0, 0, time.Local)
	}
	if _, err := application.SetStartDate("2026-08-03"); err != nil {
		t.Fatal(err)
	}
	if _, err := application.SetDone("Test", true); err != nil {
		t.Fatal(err)
	}

	var progress Progress
	progressFile := activeProgressFile(filepath.Join(directory, "data", "progress"), application.config.Profiles[0])
	if err := loadJSON(progressFile, &progress); err != nil {
		t.Fatal(err)
	}
	if !progress.History["2026-08-03"]["Test"].Done {
		t.Fatal("wykonana sesja nie została zapisana")
	}
}

func TestBreakCountdownSurvivesRestartAndFinalPartClosesProgram(t *testing.T) {
	directory := t.TempDir()
	finishedFirstAt := time.Date(2026, time.August, 31, 19, 0, 0, 0, time.Local)

	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time { return finishedFirstAt }
	config := Config{StartDate: "2026-08-31", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Ostatni dzień", DurationDays: 1, Mode: "interval",
			Program: "Program A", Time: "5 min", DeviceSteps: []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 300}},
			Scheduling: SessionScheduling{ProgramID: "program_a", Repetitions: 2, BreakBetweenMinutes: 20},
		}},
	}}}
	snapshot, err := application.SaveConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	first := planWithRepeat(snapshot.Today, 1)
	if first.SessionID == "" {
		t.Fatalf("brak pierwszej części sesji: %#v", snapshot.Today)
	}
	if _, err := application.SetSessionDone(first.SessionID, true); err != nil {
		t.Fatal(err)
	}

	// Symulacja zamknięcia aplikacji i ponownego uruchomienia po 10 minutach.
	restarted, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	restarted.now = func() time.Time {
		return time.Date(2026, time.August, 31, 19, 10, 0, 0, time.Local)
	}
	snapshot = restarted.Snapshot()
	second := planWithRepeat(snapshot.Today, 2)
	if second.Available {
		t.Fatalf("druga część nie może być dostępna przed końcem zapisanej przerwy: %#v", second)
	}
	wantAvailableAt := time.Date(2026, time.August, 31, 19, 20, 0, 0, time.Local).Format(time.RFC3339)
	if second.AvailableAt != wantAvailableAt {
		t.Fatalf("odliczanie po restarcie kończy się o %q, oczekiwano %q", second.AvailableAt, wantAvailableAt)
	}
	if completion := snapshot.Progress.Completions[first.SessionID]; completion.CompletedAt != finishedFirstAt.Format(time.RFC3339) {
		t.Fatalf("godzina pierwszego zabiegu nie przetrwała restartu: %#v", completion)
	}

	restarted.now = func() time.Time {
		return time.Date(2026, time.August, 31, 19, 20, 0, 0, time.Local)
	}
	snapshot = restarted.Snapshot()
	second = planWithRepeat(snapshot.Today, 2)
	if !second.Available {
		t.Fatalf("druga część powinna odblokować się dokładnie o 19:20: %#v", second)
	}
	if _, err := restarted.SetSessionDone(second.SessionID, true); err != nil {
		t.Fatal(err)
	}
	snapshot = restarted.Snapshot()
	if len(snapshot.Config.Profiles) != 0 || len(snapshot.Archive) != 1 {
		t.Fatalf("po ostatniej części zakończony program powinien zniknąć z aktywnego planu i trafić do archiwum: %#v", snapshot)
	}
}

func TestLegacyRootDataMigratesToDataDirectory(t *testing.T) {
	directory := t.TempDir()
	legacyProfiles := `{"start_date":"2026-08-03","profiles":[]}`
	legacyProgress := `{"start_date":"2026-08-03","history":{}}`
	if err := os.WriteFile(filepath.Join(directory, "profiles.json"), []byte(legacyProfiles), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "progress.json"), []byte(legacyProgress), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewApplication(directory); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "profiles.json")); err != nil {
		t.Fatalf("profiles.json nie został zmigrowany: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "progress")); err != nil {
		t.Fatalf("katalog progresu nie został utworzony: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "backups", "progress.json.migrated.bak")); err != nil {
		t.Fatalf("stary progress.json nie trafił do lokalnej kopii migracyjnej: %v", err)
	}
	for _, name := range []string{"profiles.json", "progress.json"} {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("stary plik %s nadal jest w katalogu głównym", name)
		}
	}
}

func TestStartingAgainArchivesOldRunAndStartsClean(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, time.August, 3, 12, 0, 0, 0, time.Local)
	}
	config := Config{StartDate: "2026-08-03", Profiles: []Profile{{
		ID: "profil_test", Name: "Test", Phases: []Phase{{
			ID: "faza_test", Name: "Start", DurationDays: 7, Mode: "interval", Program: "Program A", Time: "5 min",
			Scheduling: SessionScheduling{ProgramID: "program_a"},
		}},
	}}}
	snapshot, err := application.SaveConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = application.SetStartDate("2026-08-03")
	if err != nil {
		t.Fatal(err)
	}
	oldRunID := snapshot.Config.Profiles[0].RunID
	if _, err := application.SetDone("Test", true); err != nil {
		t.Fatal(err)
	}

	snapshot, err = application.SetStartDate("2026-08-03")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Config.Profiles[0].RunID == oldRunID {
		t.Fatal("nowy przebieg zachował stare run_id")
	}
	if len(snapshot.Today) == 0 || snapshot.Today[0].Done {
		t.Fatalf("nowy przebieg odziedziczył wykonanie: %#v", snapshot.Today)
	}
	if len(snapshot.Archive) != 1 {
		t.Fatalf("stary przebieg nie trafił do archiwum: %#v", snapshot.Archive)
	}
	if snapshot.Archive[0].Progress == nil || len(snapshot.Archive[0].Progress.Completions) != 1 {
		t.Fatalf("podgląd archiwum nie zawiera zapisanego wykonania: %#v", snapshot.Archive[0].Progress)
	}
	archiveFolder := filepath.Join(directory, "data", "archive", snapshot.Archive[0].ArchiveID)
	for _, name := range []string{"profile.json", "progress.json"} {
		if _, err := os.Stat(filepath.Join(archiveFolder, name)); err != nil {
			t.Fatalf("brak %s w folderze starego przebiegu: %v", name, err)
		}
	}
	newProgressFile := activeProgressFile(filepath.Join(directory, "data", "progress"), snapshot.Config.Profiles[0])
	if _, err := os.Stat(newProgressFile); err != nil {
		t.Fatalf("brak osobnego progresu nowego przebiegu: %v", err)
	}
}
