package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersonsContextAndAIImport(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	}
	added, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "persons.json")); err != nil {
		t.Fatalf("nie utworzono persons.json: %v", err)
	}
	context, err := application.GenerateAIContext(AIContextRequest{PersonID: added.PersonID, Mode: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(context, "**person_id:** `"+added.PersonID+"`") || !strings.Contains(context, "frequency_hz") {
		t.Fatalf("kontekst nie zawiera ID osoby albo formatu częstotliwości:\n%s", context)
	}
	if !strings.Contains(context, "# Osoba: Maks") {
		t.Fatalf("kontekst nie zawiera nagłówka osoby:\n%s", context)
	}
	for _, section := range []string{"## Skrót wykonanych programów", "## Format odpowiedzi", "## Zasady formatu"} {
		if !strings.Contains(context, section) {
			t.Fatalf("kontekst nie zawiera sekcji %q:\n%s", section, context)
		}
	}

	raw := `{
  "person_id": "` + added.PersonID + `",
  "programs": [{
    "id": "test_30k",
    "name": "Test 30 kHz",
    "steps": [{"frequency_hz": 30000, "duration_seconds": 420}],
    "min_gap_minutes": 60
  }],
  "phases": [{
    "name": "Start",
    "days": 7,
    "every_days": 2,
    "program": "test_30k"
  }]
}`
	preview, err := application.PreviewAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Persons) != 1 || preview.Persons[0].PersonID != added.PersonID || preview.Persons[0].ProgramCount != 1 || preview.Persons[0].TotalDays != 7 {
		t.Fatalf("nieprawidłowy podgląd: %#v", preview)
	}
	snapshot, err := application.ApplyAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Config.Profiles) != 1 || snapshot.Config.Profiles[0].PersonID != added.PersonID {
		t.Fatalf("profil nie został przypisany do osoby: %#v", snapshot.Config.Profiles)
	}
	step := snapshot.Config.Profiles[0].Phases[0].DeviceSteps[0]
	if step.FrequencyMilliHz != 30_000_000 || step.DurationSeconds != 420 {
		t.Fatalf("częstotliwość Hz nie została poprawnie przeliczona: %#v", step)
	}
	if snapshot.Config.Profiles[0].RunID == "" || snapshot.Config.Profiles[0].StartDate != "2026-08-04" {
		t.Fatalf("aplikacja nie nadała przebiegu i daty startu: %#v", snapshot.Config.Profiles[0])
	}
}

func TestPausedSessionPersistsAndChangesDailyRemainingTime(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	}
	config := Config{StartDate: "2026-08-04", Profiles: []Profile{{
		ID: "test", PersonID: "test", Name: "Test", StartDate: "2026-08-04",
		Phases: []Phase{{
			Name: "Start", DurationDays: 1, Mode: "interval", Program: "Test", Time: "7 min",
			DeviceSteps: []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 420}},
			Scheduling:  SessionScheduling{ProgramID: "test", Repetitions: 1},
		}},
	}}}
	snapshot, err := application.SaveConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Today) != 1 {
		t.Fatalf("brak dzisiejszej sesji: %#v", snapshot.Today)
	}
	sessionID := snapshot.Today[0].SessionID
	snapshot, err = application.SavePausedSession(DevicePauseState{
		SessionID: sessionID, RemainingSteps: []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 180}}, RemainingSeconds: 180,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Today[0].Paused || snapshot.Today[0].RemainingSeconds != 180 || snapshot.TodayRemainingSeconds != 180 {
		t.Fatalf("pauza nie zmieniła pozostałego czasu: %#v / %d", snapshot.Today[0], snapshot.TodayRemainingSeconds)
	}
	if _, err := application.SetSessionDone(sessionID, true); err != nil {
		t.Fatal(err)
	}
	if _, exists := application.progress.PausedSessions[sessionID]; exists {
		t.Fatal("wykonana sesja nadal ma zapis pauzy")
	}
}

func TestGenerateAIContextMultiplePersons(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	first, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.AddPerson("Anna", "")
	if err != nil {
		t.Fatal(err)
	}
	context, err := application.GenerateAIContext(AIContextRequest{PersonIDs: []string{first.PersonID, second.PersonID}, Mode: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(context, "**person_id:** `"+first.PersonID+"`") || !strings.Contains(context, "**person_id:** `"+second.PersonID+"`") {
		t.Fatalf("kontekst nie zawiera obu person_id:\n%s", context)
	}
	if !strings.Contains(context, "# Osoba: Maks") || !strings.Contains(context, "# Osoba: Anna") {
		t.Fatalf("kontekst nie zawiera nagłówków obu osób:\n%s", context)
	}
	if strings.Count(context, "# Osoba: ") != 2 || strings.Count(context, "## Format odpowiedzi") != 2 {
		t.Fatalf("kontekst nie zawiera dwóch pełnych bloków osób:\n%s", context)
	}
	if !strings.Contains(context, "JEDNĄ tablicę") {
		t.Fatalf("kontekst wielu osób nie informuje o odpowiedzi w tablicy JSON:\n%s", context)
	}
	if !strings.Contains(context, "\n---\n") {
		t.Fatalf("kontekst wielu osób nie zawiera separatora markdown:\n%s", context)
	}
}

func TestApplyAIProfileImportsManyPersonsFromOneArray(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	}
	first, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.AddPerson("Anna", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := `[
  {
    "person_id": "` + first.PersonID + `",
    "programs": [{
      "id": "first_30k",
      "name": "Pierwszy 30 kHz",
      "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]
    }],
    "phases": [{"name": "Start", "days": 7, "program": "first_30k"}]
  },
  {
    "person_id": "` + second.PersonID + `",
    "programs": [{
      "id": "second_40k",
      "name": "Drugi 40 kHz",
      "steps": [{"frequency_hz": 40000, "duration_seconds": 300}]
    }],
    "phases": [{"name": "Start", "days": 5, "program": "second_40k"}]
  }
]`
	preview, err := application.PreviewAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Persons) != 2 || preview.Persons[0].PersonID != first.PersonID || preview.Persons[1].PersonID != second.PersonID {
		t.Fatalf("podgląd nie obejmuje obu osób: %#v", preview)
	}
	snapshot, err := application.ApplyAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Config.Profiles) != 2 {
		t.Fatalf("oczekiwano dwóch profili: %#v", snapshot.Config.Profiles)
	}
	seen := map[string]bool{}
	for _, profile := range snapshot.Config.Profiles {
		seen[profile.PersonID] = true
	}
	if !seen[first.PersonID] || !seen[second.PersonID] {
		t.Fatalf("profile nie zostały przypisane do obu osób: %#v", seen)
	}
	for _, person := range snapshot.Persons {
		if !person.Active {
			t.Fatalf("import nie ustawił osoby %s jako aktywnej: %#v", person.ID, person)
		}
	}
}

func TestApplyAIProfileRejectsArrayWithUnknownPerson(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := `[
  {"person_id": "` + added.PersonID + `", "programs": [{"id": "p1", "name": "P", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}], "phases": [{"name": "S", "days": 3, "program": "p1"}]},
  {"person_id": "kogos_nie_ma", "programs": [{"id": "p2", "name": "P2", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}], "phases": [{"name": "S", "days": 3, "program": "p2"}]}
]`
	if _, err := application.ApplyAIProfile(raw); err == nil {
		t.Fatal("import z nieznaną osobą powinien się nie udać")
	} else if !strings.Contains(err.Error(), "kogos_nie_ma") || !strings.Contains(err.Error(), "(element 2)") {
		t.Fatalf("błąd nie wskazuje osoby i elementu: %v", err)
	}
	if len(application.config.Profiles) != 0 {
		t.Fatalf("nieudany import nie może nic zapisać: %#v", application.config.Profiles)
	}
}

func TestApplyAIProfileRejectsArrayWithDuplicatePerson(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := `[
  {"person_id": "` + added.PersonID + `", "programs": [{"id": "p1", "name": "P", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}], "phases": [{"name": "S", "days": 3, "program": "p1"}]},
  {"person_id": "` + added.PersonID + `", "programs": [{"id": "p2", "name": "P2", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}], "phases": [{"name": "S", "days": 3, "program": "p2"}]}
]`
	if _, err := application.ApplyAIProfile(raw); err == nil || !strings.Contains(err.Error(), "występuje więcej niż raz") {
		t.Fatalf("duplikat osoby powinien zostać odrzucony: %v", err)
	}
}

func TestStopSessionRecordsPartialRun(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local)
	}
	added, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := `{
  "person_id": "` + added.PersonID + `",
  "phases": [{"name": "Start", "days": 7, "frequency_hz": 30000, "duration_seconds": 420}]
}`
	if _, err := application.ApplyAIProfile(raw); err != nil {
		t.Fatal(err)
	}
	snapshot := application.Snapshot()
	sessionID := snapshot.Today[0].SessionID
	// Zatrzymaj po 204 sekundach: wykonano 204, zostało 216 z 420.
	snapshot, err = application.SavePausedSession(DevicePauseState{
		SessionID: sessionID, RemainingSteps: []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 216}}, RemainingSeconds: 216,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	paused := application.progress.PausedSessions[sessionID]
	if !paused.Recorded {
		t.Fatalf("zatrzymanie powinno oznaczyć pauzę jako zarejestrowaną: %#v", paused)
	}
	runs := application.progress.PartialRuns
	if len(runs) != 1 {
		t.Fatalf("oczekiwano 1 wpisu częściowego, jest %d", len(runs))
	}
	run := runs[0]
	if run.DoneSeconds != 204 || run.TotalSeconds != 420 || run.RemainingSeconds != 216 {
		t.Fatalf("nieprawidłowy wpis częściowy: %#v", run)
	}
	if run.SessionID != sessionID || run.ProfileName != "Maks" || run.Therapy != "30 kHz" {
		t.Fatalf("nieprawidłowe pola wpisu: %#v", run)
	}
	if !snapshot.Today[0].Paused || !snapshot.Today[0].PausedRecorded {
		t.Fatalf("sesja ma pozostać wstrzymana na liście z flagą rejestracji: %#v", snapshot.Today[0])
	}
}

func TestAIImportWarnsWhenCooldownIsImplicitlyZero(t *testing.T) {
	warnings := aiSchedulingWarnings(aiProfileInput{
		Programs: []aiProgram{{ID: "clark", Name: "Clark"}},
		Phases:   []aiPhase{{Name: "Start", Program: "clark"}},
	})
	if len(warnings) != 1 || !strings.Contains(warnings[0], "cooldown_after_minutes") {
		t.Fatalf("brak ostrzeżenia o pominiętym cooldownie: %#v", warnings)
	}
	zero := 0
	warnings = aiSchedulingWarnings(aiProfileInput{
		Programs: []aiProgram{{ID: "regeneracja", Name: "Regeneracja", CooldownAfterMinutes: &zero}},
		Phases:   []aiPhase{{Name: "Start", Program: "regeneracja"}},
	})
	if len(warnings) != 0 {
		t.Fatalf("jawne zero nie powinno wyglądać jak pominięte pole: %#v", warnings)
	}
}
