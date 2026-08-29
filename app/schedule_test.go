package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestIntervalSchedule(t *testing.T) {
	profile := Profile{
		Name: "Test",
		Phases: []Phase{{
			Name:         "Start",
			DurationDays: 4,
			Mode:         "interval",
			IntervalGap:  1,
			Program:      "30 kHz",
			Time:         "7 min",
			Note:         "3x",
		}},
	}
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)

	first := planForDate(profile, start, start)
	if first.Status != "session" || first.Program != "30 kHz" || first.Repetitions != "3x" {
		t.Fatalf("nieprawidłowy pierwszy dzień: %#v", first)
	}
	second := planForDate(profile, start.AddDate(0, 0, 1), start)
	if second.Status != "break" {
		t.Fatalf("drugi dzień powinien być przerwą: %#v", second)
	}
	third := planForDate(profile, start.AddDate(0, 0, 2), start)
	if third.Status != "session" {
		t.Fatalf("trzeci dzień powinien zawierać sesję: %#v", third)
	}
}

func TestStructuredRepetitionsOverrideLegacyText(t *testing.T) {
	profile := Profile{
		Name: "Test",
		Phases: []Phase{{
			Name:         "Start",
			DurationDays: 1,
			Mode:         "interval",
			Program:      "30 kHz",
			Time:         "2x 7 min",
			Note:         "dwa razy",
			Scheduling:   SessionScheduling{Repetitions: 3},
		}},
	}
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)
	plan := planForDate(profile, start, start)
	if plan.Repetitions != "3x" {
		t.Fatalf("harmonogram powinien brać liczbę powtórzeń z scheduling, otrzymano %q", plan.Repetitions)
	}
}

func TestWeeklyScheduleUsesRequestedDate(t *testing.T) {
	profile := Profile{
		Name: "Test",
		Phases: []Phase{{
			Name:         "Tydzień",
			DurationDays: 14,
			Mode:         "weekly",
			Schedule: map[string]DailyPlan{
				"Monday": {Frequency: "880 Hz", Time: "10 min"},
			},
		}},
	}
	monday := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.Local)
	if plan := planForDate(profile, monday, monday); plan.Status != "session" || plan.Program != "880 Hz" {
		t.Fatalf("poniedziałek powinien zawierać sesję: %#v", plan)
	}
	if plan := planForDate(profile, monday.AddDate(0, 0, 1), monday); plan.Status != "break" {
		t.Fatalf("wtorek powinien być wolny: %#v", plan)
	}
}

func TestLegacyBooleanHistoryEntry(t *testing.T) {
	var entry HistoryEntry
	if err := json.Unmarshal([]byte("true"), &entry); err != nil {
		t.Fatal(err)
	}
	if !entry.Done {
		t.Fatal("stary wpis true powinien zostać odczytany jako wykonany")
	}
}

func TestFrequencyDatabaseIsAvailable(t *testing.T) {
	entries := loadFrequencyDatabase()
	if len(entries) < 100 {
		t.Fatalf("oczekiwano pełnej bazy, odczytano tylko %d pozycji", len(entries))
	}
}
