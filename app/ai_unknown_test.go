package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func newUnknownFieldsApplication(t *testing.T) (*Application, string) {
	t.Helper()
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
	return application, added.PersonID
}

func unknownFieldsJSON(body, personID string) string {
	return strings.ReplaceAll(body, "%PERSON%", personID)
}

// Nieznane pole na każdym poziomie: root, program, krok programu, faza,
// sesja w planie tygodniowym.
const unknownEveryLevelResponse = `{
  "person_id": "%PERSON%",
  "wersja": 3,
  "programs": [
    {
      "id": "program_a",
      "name": "Program A",
      "description": "opis spoza schematu",
      "steps": [{"frequency_hz": 30000, "duration_seconds": 420, "amplitude": 5}]
    }
  ],
  "phases": [
    {
      "name": "Start",
      "days": 7,
      "every_days": 2,
      "program": "program_a",
      "priority": "high"
    },
    {
      "name": "Tydzień",
      "days": 14,
      "week": {
        "Monday": "program_a",
        "Friday": {"program": "program_a", "repeat": 2, "intensity": 3}
      }
    }
  ]
}`

func TestPreviewReportsUnknownFieldsAtEveryLevel(t *testing.T) {
	application, personID := newUnknownFieldsApplication(t)
	preview, err := application.PreviewAIProfile(unknownFieldsJSON(unknownEveryLevelResponse, personID))
	if err != nil {
		t.Fatalf("import z nieznanymi polami został odrzucony: %v", err)
	}
	expected := []string{
		"phases[0].priority",
		"phases[1].week.Friday.intensity",
		"programs[0].description",
		"programs[0].steps[0].amplitude",
		"wersja",
	}
	if !reflect.DeepEqual(preview.Persons[0].UnknownFields, expected) {
		t.Fatalf("nieprawidłowa lista nieznanych pól:\n%#v\noczekiwano:\n%#v", preview.Persons[0].UnknownFields, expected)
	}
	if preview.Persons[0].PhaseCount != 2 || preview.Persons[0].TotalDays != 21 {
		t.Fatalf("nieznane pola zaburzyły podgląd: %#v", preview)
	}
	if _, err := application.ApplyAIProfile(unknownFieldsJSON(unknownEveryLevelResponse, personID)); err != nil {
		t.Fatalf("import z nieznanymi polami nie przeszedł: %v", err)
	}
}

// Główny scenariusz: literówka `repeats` zamiast `repeat`.
const typoRepeatsResponse = `{
  "person_id": "%PERSON%",
  "programs": [
    {"id": "program_a", "name": "Program A", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}
  ],
  "phases": [
    {"name": "Start", "days": 7, "program": "program_a", "repeats": 3}
  ]
}`

func TestTypoRepeatsImportsWithDefaultAndIsReported(t *testing.T) {
	application, personID := newUnknownFieldsApplication(t)
	raw := unknownFieldsJSON(typoRepeatsResponse, personID)
	preview, err := application.PreviewAIProfile(raw)
	if err != nil {
		t.Fatalf("literówka w nazwie pola wywaliła import: %v", err)
	}
	if !reflect.DeepEqual(preview.Persons[0].UnknownFields, []string{"phases[0].repeats"}) {
		t.Fatalf("literówka nie trafiła na listę nieznanych pól: %#v", preview.Persons[0].UnknownFields)
	}
	_, profile, err := application.parseAIProfileLocked(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.Phases[0].Scheduling.Repetitions; got != 1 {
		t.Fatalf("repeat powinien mieć wartość domyślną 1, ma %d", got)
	}
}

// Czysty JSON — żadnych fałszywych trafień, w tym w kluczach mapy `week`.
const cleanFullResponse = `{
  "person_id": "%PERSON%",
  "programs": [
    {
      "id": "program_a",
      "name": "Program A",
      "steps": [{"frequency_hz": 30000, "duration_seconds": 420}],
      "min_gap_minutes": 60,
      "cooldown_after_minutes": 30
    },
    {
      "id": "program_b",
      "name": "Program B",
      "steps": [{"frequency_hz": 7.83, "duration_seconds": 600}]
    }
  ],
  "same_day_pairs": [["program_a", "program_b"]],
  "phases": [
    {"name": "Start", "days": 7, "every_days": 2, "program": "program_a", "repeat": 2, "break_minutes": 15, "note": "uwaga"},
    {
      "name": "Tydzień",
      "days": 14,
      "week": {
        "Monday": "program_a",
        "Tuesday": "program_b",
        "Wednesday": "program_a",
        "Thursday": "program_b",
        "Friday": {"program": "program_a", "repeat": 2, "break_minutes": 20, "note": "wieczorem"},
        "Saturday": "program_b",
        "Sunday": {"program": "program_a"}
      }
    }
  ]
}`

func TestCleanResponseReportsNoUnknownFields(t *testing.T) {
	application, personID := newUnknownFieldsApplication(t)
	preview, err := application.PreviewAIProfile(unknownFieldsJSON(cleanFullResponse, personID))
	if err != nil {
		t.Fatalf("poprawny JSON został odrzucony: %v", err)
	}
	if len(preview.Persons[0].UnknownFields) != 0 {
		t.Fatalf("fałszywe trafienia dla poprawnego JSON-a: %#v", preview.Persons[0].UnknownFields)
	}
}

// Nazwy pól są dopasowywane bez rozróżniania wielkości liter — tak samo jak
// robi to encoding/json — więc `Days` nie jest polem nieznanym.
func TestUnknownFieldsIgnoreLetterCaseLikeEncodingJSON(t *testing.T) {
	raw := `{"person_id": "x", "programs": [{"ID": "a", "Name": "A", "steps": []}], "phases": [{"name": "S", "Days": 3}]}`
	if got := collectUnknownFields(raw, aiProfileInput{}); len(got) != 0 {
		t.Fatalf("dopasowanie wielkości liter dało fałszywe trafienia: %#v", got)
	}
}

func TestUnknownFieldsToleratesBrokenJSON(t *testing.T) {
	if got := collectUnknownFields("{nie-json", aiProfileInput{}); got != nil {
		t.Fatalf("dla niepoprawnego JSON-a oczekiwano nil, jest %#v", got)
	}
}
