package main

import (
	"strings"
	"testing"
	"time"
)

func newFrequencyTestApp(t *testing.T) (*Application, string) {
	t.Helper()
	application, err := NewApplication(t.TempDir())
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
	return application, added.PersonID
}

func applyFrequencyImport(t *testing.T, app *Application, personID, raw string) Snapshot {
	t.Helper()
	if _, err := app.PreviewAIProfile(raw); err != nil {
		t.Fatalf("podgląd: %v", err)
	}
	snapshot, err := app.ApplyAIProfile(raw)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return snapshot
}

func TestAIFrequencyOnlyInterval(t *testing.T) {
	app, personID := newFrequencyTestApp(t)
	raw := `{
  "person_id": "` + personID + `",
  "phases": [
    {
      "name": "Start",
      "days": 7,
      "every_days": 2,
      "frequency_hz": 30000,
      "duration_seconds": 420
    }
  ]
}`
	snapshot := applyFrequencyImport(t, app, personID, raw)
	profile := snapshot.Config.Profiles[0]
	if len(profile.Phases) != 1 {
		t.Fatalf("oczekiwano 1 fazy, jest %d", len(profile.Phases))
	}
	phase := profile.Phases[0]
	if phase.Mode != "interval" || phase.IntervalGap != 1 || phase.Program != "30 kHz" {
		t.Fatalf("nieprawidłowa faza: %#v", phase)
	}
	if len(phase.DeviceSteps) != 1 || phase.DeviceSteps[0].FrequencyMilliHz != 30_000_000 || phase.DeviceSteps[0].DurationSeconds != 420 {
		t.Fatalf("nieprawidłowe kroki: %#v", phase.DeviceSteps)
	}
	if phase.Scheduling.Repetitions != 1 || phase.Scheduling.BreakBetweenMinutes != 0 {
		t.Fatalf("oczekiwano domyślnego harmonogramu: %#v", phase.Scheduling)
	}
}

func TestAIFrequencyOnlyWeekly(t *testing.T) {
	app, personID := newFrequencyTestApp(t)
	raw := `{
  "person_id": "` + personID + `",
  "phases": [
    {
      "name": "Tydzień",
      "days": 7,
      "week": {
        "Monday": {"frequency_hz": 30000, "duration_seconds": 420},
        "Wednesday": {"steps": [{"frequency_hz": 20, "duration_seconds": 300}, {"frequency_hz": 727, "duration_seconds": 180}]},
        "Friday": {"frequency_hz": 30000, "duration_seconds": 420, "repeat": 2, "break_minutes": 30, "note": "dwa razy"}
      }
    }
  ]
}`
	preview, err := app.PreviewAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Persons[0].ProgramCount != 2 {
		t.Fatalf("oczekiwano 2 różnych sesji w podglądzie, jest %d", preview.Persons[0].ProgramCount)
	}
	snapshot := applyFrequencyImport(t, app, personID, raw)
	schedule := snapshot.Config.Profiles[0].Phases[0].Schedule
	monday := schedule["Monday"]
	if monday.Frequency != "30 kHz" || len(monday.DeviceSteps) != 1 || monday.DeviceSteps[0].DurationSeconds != 420 {
		t.Fatalf("nieprawidłowy poniedziałek: %#v", monday)
	}
	wednesday := schedule["Wednesday"]
	if len(wednesday.DeviceSteps) != 2 || wednesday.DeviceSteps[1].FrequencyMilliHz != 727_000 || wednesday.Frequency != "20 Hz + 727 Hz" {
		t.Fatalf("nieprawidłowa środa: %#v", wednesday)
	}
	friday := schedule["Friday"]
	if friday.Scheduling.Repetitions != 2 || friday.Scheduling.BreakBetweenMinutes != 30 || friday.Note != "dwa razy" {
		t.Fatalf("nieprawidłowy piątek: %#v", friday)
	}
}

func TestAIFrequencyOnlyErrors(t *testing.T) {
	app, personID := newFrequencyTestApp(t)
	thirteenSteps := `[{"frequency_hz": 1, "duration_seconds": 60}, {"frequency_hz": 2, "duration_seconds": 60}, {"frequency_hz": 3, "duration_seconds": 60}, {"frequency_hz": 4, "duration_seconds": 60}, {"frequency_hz": 5, "duration_seconds": 60}, {"frequency_hz": 6, "duration_seconds": 60}, {"frequency_hz": 7, "duration_seconds": 60}, {"frequency_hz": 8, "duration_seconds": 60}, {"frequency_hz": 9, "duration_seconds": 60}, {"frequency_hz": 10, "duration_seconds": 60}, {"frequency_hz": 11, "duration_seconds": 60}, {"frequency_hz": 12, "duration_seconds": 60}, {"frequency_hz": 13, "duration_seconds": 60}]`
	cases := []struct {
		name    string
		day     string
		wantErr string
	}{
		{"brak czasu", `{"frequency_hz": 30000}`, "duration_seconds"},
		{"brak sesji", `{}`, "program albo frequency_hz"},
		{"za dużo kroków", `{"steps": ` + thirteenSteps + `}`, "12 kroków"},
		{"zła częstotliwość", `{"frequency_hz": 0, "duration_seconds": 60}`, "0.1–4000000"},
	}
	for _, tc := range cases {
		raw := `{"person_id": "` + personID + `", "phases": [{"name": "Faza", "days": 3, "week": {"Monday": ` + tc.day + `}}]}`
		if _, err := app.ApplyAIProfile(raw); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: oczekiwano błędu z %q, jest: %v", tc.name, tc.wantErr, err)
		}
	}
}

func TestAIFullFormatStillImports(t *testing.T) {
	app, personID := newFrequencyTestApp(t)
	raw := `{
  "person_id": "` + personID + `",
  "programs": [
    {"id": "a", "name": "Sesja A", "steps": [{"frequency_hz": 10000, "duration_seconds": 300}], "min_gap_minutes": 30},
    {"id": "b", "name": "Sesja B", "steps": [{"frequency_hz": 20000, "duration_seconds": 300}]}
  ],
  "same_day_pairs": [["a", "b"]],
  "phases": [
    {"name": "Start", "days": 5, "every_days": 1, "program": "a"},
    {"name": "Week", "days": 7, "week": {"Monday": "b", "Friday": {"program": "a", "repeat": 2, "break_minutes": 15}}}
  ]
}`
	preview, err := app.PreviewAIProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Persons[0].ProgramCount != 2 {
		t.Fatalf("oczekiwano 2 programów w podglądzie, jest %d", preview.Persons[0].ProgramCount)
	}
	snapshot := applyFrequencyImport(t, app, personID, raw)
	profile := snapshot.Config.Profiles[0]
	if len(profile.Phases) != 2 || profile.Phases[0].Program != "Sesja A" || profile.Phases[0].Scheduling.MinGapMinutes != 30 {
		t.Fatalf("nieprawidłowy profil: %#v", profile)
	}
	friday := profile.Phases[1].Schedule["Friday"]
	if friday.Scheduling.ProgramID != "a" || friday.Scheduling.Repetitions != 2 || friday.Scheduling.BreakBetweenMinutes != 15 {
		t.Fatalf("nieprawidłowy piątek: %#v", friday)
	}
	if len(friday.Scheduling.SameDayWith) != 1 || friday.Scheduling.SameDayWith[0] != "b" {
		t.Fatalf("brak relacji same-day: %#v", friday.Scheduling)
	}
}

func TestAIMixedProgramsAndFrequency(t *testing.T) {
	app, personID := newFrequencyTestApp(t)
	raw := `{
  "person_id": "` + personID + `",
  "programs": [
    {"id": "a", "name": "Sesja A", "steps": [{"frequency_hz": 10000, "duration_seconds": 300}]}
  ],
  "phases": [
    {"name": "Start", "days": 5, "program": "a"},
    {"name": "Week", "days": 7, "week": {"Saturday": {"frequency_hz": 30000, "duration_seconds": 420}}}
  ]
}`
	snapshot := applyFrequencyImport(t, app, personID, raw)
	profile := snapshot.Config.Profiles[0]
	if profile.Phases[0].Program != "Sesja A" || profile.Phases[1].Schedule["Saturday"].Frequency != "30 kHz" {
		t.Fatalf("nieprawidłowy import mieszany: %#v", profile)
	}
}
