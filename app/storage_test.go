package main

import "testing"

func TestValidationAcceptsSharedProgramIDWithDifferentRepetitions(t *testing.T) {
	config := Config{Profiles: []Profile{{
		ID: "profil_a", Name: "Profil A", Phases: []Phase{
			{ID: "faza_solo", Name: "Solo", DurationDays: 1, Mode: "interval", Program: "Program A", Scheduling: SessionScheduling{
				ProgramID: "program_a", Repetitions: 1, MinGapMinutes: 2880, CooldownAfterMinutes: 1440,
			}},
			{ID: "faza_seria", Name: "Seria", DurationDays: 1, Mode: "interval", Program: "Program A — seria", Scheduling: SessionScheduling{
				ProgramID: "program_a", Repetitions: 3, BreakBetweenMinutes: 20, MinGapMinutes: 2880, CooldownAfterMinutes: 1440,
			}},
		},
	}}}
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("wspólne ID dla solo i serii powinno być poprawne: %v", err)
	}
}

func TestValidationRejectsOneSidedCompatibility(t *testing.T) {
	config := Config{Profiles: []Profile{{
		ID: "profil_a", Name: "Profil A", Phases: []Phase{
			{ID: "faza_a", Name: "A", DurationDays: 1, Mode: "interval", Program: "Program A", Scheduling: SessionScheduling{ProgramID: "program_a", SameDayWith: []string{"program_b"}}},
			{ID: "faza_b", Name: "B", DurationDays: 1, Mode: "interval", Program: "Program B", Scheduling: SessionScheduling{ProgramID: "program_b"}},
		},
	}}}
	if _, err := validateConfig(&config, validationModeSave); err == nil {
		t.Fatal("jednostronna zgodność powinna zostać odrzucona")
	}
	config.Profiles[0].Phases[1].Scheduling.SameDayWith = []string{"program_a"}
	if _, err := validateConfig(&config, validationModeSave); err != nil {
		t.Fatalf("wzajemna zgodność powinna zostać zaakceptowana: %v", err)
	}
}

func TestValidationRejectsUnknownProgramReferenceAndDuplicateIDs(t *testing.T) {
	unknown := Config{Profiles: []Profile{{
		ID: "profil_a", Name: "Profil A", Phases: []Phase{{
			ID: "faza_a", Name: "A", DurationDays: 1, Mode: "interval", Program: "Program A",
			Scheduling: SessionScheduling{ProgramID: "program_a", SameDayWith: []string{"brak_programu"}},
		}},
	}}}
	if _, err := validateConfig(&unknown, validationModeSave); err == nil {
		t.Fatal("odwołanie do nieistniejącego programu powinno zostać odrzucone")
	}

	duplicate := Config{Profiles: []Profile{
		{ID: "ten_sam", Name: "Pierwszy"},
		{ID: "ten_sam", Name: "Drugi"},
	}}
	if _, err := validateConfig(&duplicate, validationModeSave); err == nil {
		t.Fatal("powtórzony identyfikator profilu powinien zostać odrzucony")
	}
}
