package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// contradictoryConfig: min_gap 4320 min przy planie co 2 dni (2880 min) —
// plan, którego nigdy nie da się wykonać na czas.
func contradictoryConfig() Config {
	return Config{StartDate: "2026-08-04", Profiles: []Profile{{
		ID: "profil_a", PersonID: "profil_a", Name: "Profil A", StartDate: "2026-08-04", RunID: "run_test",
		Phases: []Phase{{
			ID: "faza_a", Name: "Faza A", DurationDays: 30, Mode: "interval", Program: "Program A", IntervalGap: 1,
			Scheduling: SessionScheduling{ProgramID: "program_a", Repetitions: 1, MinGapMinutes: 4320},
		}},
	}}}
}

// realUserConfig odwzorowuje dane użytkownika: min_gap == okres == 2880 min.
// To jest przypadek graniczny, który MUSI przechodzić w obu trybach.
func realUserConfig() Config {
	phase := func(id, name, program string) Phase {
		return Phase{
			ID: id, Name: name, DurationDays: 28, Mode: "interval", Program: program, IntervalGap: 1,
			Scheduling: SessionScheduling{ProgramID: readableID(program), Repetitions: 1, MinGapMinutes: 2880, CooldownAfterMinutes: 1440},
		}
	}
	return Config{StartDate: "2026-08-04", Profiles: []Profile{{
		ID: "maks", PersonID: "maks", Name: "Maks", StartDate: "2026-08-04", RunID: "run_test",
		Phases: []Phase{
			phase("faza_1", "Faza 1", "Program A"),
			phase("faza_2", "Faza 2", "Program B"),
			phase("faza_3", "Faza 3", "Program C"),
		},
	}}}
}

func writeProfiles(t *testing.T, directory string, config Config) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(directory, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "data", "profiles.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestContradictoryConfigLoadsWithWarningInsteadOfBlockingStartup(t *testing.T) {
	directory := t.TempDir()
	writeProfiles(t, directory, contradictoryConfig())

	application, err := NewApplication(directory)
	if err != nil {
		t.Fatalf("sprzeczna konfiguracja NIE MOŻE blokować startu aplikacji: %v", err)
	}
	warnings := application.LoadWarnings()
	if len(warnings) != 1 {
		t.Fatalf("oczekiwano jednego ostrzeżenia, jest %d: %v", len(warnings), warnings)
	}
	for _, fragment := range []string{"Faza A", "Profil A", "4320", "2880", "popraw w edytorze"} {
		if !strings.Contains(warnings[0], fragment) {
			t.Fatalf("ostrzeżenie nie zawiera %q: %s", fragment, warnings[0])
		}
	}
	if len(application.config.Profiles) != 1 {
		t.Fatal("profil użytkownika zniknął mimo poprawnego startu")
	}
}

func TestContradictoryConfigIsRejectedOnSave(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	_, err = application.SaveConfig(contradictoryConfig())
	if err == nil {
		t.Fatal("zapis sprzecznej konfiguracji musi zostać odrzucony")
	}
	for _, fragment := range []string{"minimalny odstęp", "4320", "2880"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("błąd zapisu nie zawiera %q: %v", fragment, err)
		}
	}
}

func TestStructuralValidationBlocksInBothModes(t *testing.T) {
	cases := map[string]func() Config{
		"nieznany tryb": func() Config {
			config := realUserConfig()
			config.Profiles[0].Phases[0].Mode = "kosmiczny"
			return config
		},
		"brak programu w fazie interwałowej": func() Config {
			config := realUserConfig()
			config.Profiles[0].Phases[0].Program = ""
			return config
		},
		"powtórzony identyfikator profilu": func() Config {
			config := realUserConfig()
			second := config.Profiles[0]
			second.Name = "Inny"
			config.Profiles = append(config.Profiles, second)
			return config
		},
		"odwołanie do nieistniejącego programu": func() Config {
			config := realUserConfig()
			config.Profiles[0].Phases[0].Scheduling.SameDayWith = []string{"nie_ma_takiego"}
			return config
		},
		"faza bez nazwy": func() Config {
			config := realUserConfig()
			config.Profiles[0].Phases[0].Name = ""
			return config
		},
	}
	for name, build := range cases {
		for modeName, mode := range map[string]validationMode{"load": validationModeLoad, "save": validationModeSave} {
			config := build()
			if _, err := validateConfig(&config, mode); err == nil {
				t.Fatalf("%s: tryb %s musi zostać odrzucony — bez tego runtime i tak nie ruszy", name, modeName)
			}
		}
	}
}

func TestRealUserConfigAtBoundaryLoadsAndSaves(t *testing.T) {
	directory := t.TempDir()
	writeProfiles(t, directory, realUserConfig())

	application, err := NewApplication(directory)
	if err != nil {
		t.Fatalf("realne dane użytkownika (min_gap == okres == 2880) muszą się wczytać: %v", err)
	}
	if warnings := application.LoadWarnings(); len(warnings) != 0 {
		t.Fatalf("dane na granicy nie mogą generować ostrzeżeń: %v", warnings)
	}
	application.now = func() time.Time { return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local) }
	if _, err := application.SaveConfig(realUserConfig()); err != nil {
		t.Fatalf("realne dane użytkownika muszą dać się zapisać: %v", err)
	}
}

func TestApplyAIProfileStillRejectsContradictoryPlan(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time { return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.Local) }
	added, err := application.AddPerson("Maks", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := `{
  "person_id": "` + added.PersonID + `",
  "programs": [{
    "id": "test_30k",
    "name": "Test 30 kHz",
    "steps": [{"frequency_hz": 30000, "duration_seconds": 420}],
    "min_gap_minutes": 4320
  }],
  "phases": [{
    "name": "Start",
    "days": 30,
    "every_days": 2,
    "program": "test_30k"
  }]
}`
	if _, err := application.ApplyAIProfile(raw); err == nil {
		t.Fatal("import z AI musi odrzucać sprzeczne plany")
	}
}
