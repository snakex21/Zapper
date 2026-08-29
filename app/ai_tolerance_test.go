package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func newToleranceApplication(t *testing.T) (*Application, string) {
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

// parseToleranceProfile parsuje odpowiedź AI i zeruje pola losowe (run_id),
// żeby dwa profile dały się porównać strukturalnie.
func parseToleranceProfile(t *testing.T, application *Application, raw string) Profile {
	t.Helper()
	_, profile, err := application.parseAIProfileLocked(raw)
	if err != nil {
		t.Fatalf("parser odrzucił poprawny JSON: %v\n%s", err, raw)
	}
	profile.RunID = ""
	return profile
}

// pełny, „stary” format: wszystkie pola jawne, same_day_pairs obustronnie.
const legacyFullAIResponse = `{
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
      "steps": [{"frequency_hz": 7.83, "duration_seconds": 600}],
      "min_gap_minutes": 0,
      "cooldown_after_minutes": 0
    }
  ],
  "same_day_pairs": [["program_a", "program_b"], ["program_b", "program_a"]],
  "phases": [
    {
      "name": "Start",
      "days": 7,
      "every_days": 2,
      "program": "program_a",
      "repeat": 1,
      "break_minutes": 0,
      "note": ""
    },
    {
      "name": "Tydzien",
      "days": 28,
      "week": {
        "Monday": {"program": "program_a", "repeat": 1, "break_minutes": 0, "note": ""},
        "Friday": {"program": "program_b", "repeat": 2, "break_minutes": 20, "note": "wieczorem"}
      }
    }
  ]
}`

func toleranceJSON(body, personID string) string {
	return strings.ReplaceAll(body, "%PERSON%", personID)
}

// TestLegacyFullAIResponseStillParses to test regresji: pełny format musi dać
// dokładnie ten sam profil co przed poluzowaniem parsera.
func TestLegacyFullAIResponseStillParses(t *testing.T) {
	application, personID := newToleranceApplication(t)
	profile := parseToleranceProfile(t, application, toleranceJSON(legacyFullAIResponse, personID))

	if len(profile.Phases) != 2 {
		t.Fatalf("oczekiwano 2 faz: %#v", profile.Phases)
	}
	first := profile.Phases[0]
	if first.Mode != "interval" || first.DurationDays != 7 || first.IntervalGap != 1 {
		t.Fatalf("faza interwałowa zmieniła kształt: %#v", first)
	}
	if first.Program != "Program A" || first.Time != "7 min" {
		t.Fatalf("faza interwałowa ma inny program/czas: %#v", first)
	}
	wantFirst := SessionScheduling{
		ProgramID: "program_a", Repetitions: 1, BreakBetweenMinutes: 0,
		MinGapMinutes: 60, CooldownAfterMinutes: 30, SameDayWith: []string{"program_b"},
	}
	if !reflect.DeepEqual(first.Scheduling, wantFirst) {
		t.Fatalf("scheduling fazy interwałowej: %#v, oczekiwano %#v", first.Scheduling, wantFirst)
	}
	wantSteps := []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 420}}
	if !reflect.DeepEqual(first.DeviceSteps, wantSteps) {
		t.Fatalf("kroki fazy interwałowej: %#v", first.DeviceSteps)
	}

	second := profile.Phases[1]
	if second.Mode != "weekly" || len(second.Schedule) != 2 {
		t.Fatalf("faza tygodniowa zmieniła kształt: %#v", second)
	}
	friday := second.Schedule["Friday"]
	wantFriday := SessionScheduling{
		ProgramID: "program_b", Repetitions: 2, BreakBetweenMinutes: 20,
		MinGapMinutes: 0, CooldownAfterMinutes: 0, SameDayWith: []string{"program_a"},
	}
	if !reflect.DeepEqual(friday.Scheduling, wantFriday) {
		t.Fatalf("scheduling piątku: %#v, oczekiwano %#v", friday.Scheduling, wantFriday)
	}
	if friday.Note != "wieczorem" || friday.Frequency != "Program B" || friday.Time != "10 min" {
		t.Fatalf("piątek zmienił treść: %#v", friday)
	}
	monday := second.Schedule["Monday"]
	if monday.Scheduling.Repetitions != 1 || monday.Scheduling.BreakBetweenMinutes != 0 {
		t.Fatalf("poniedziałek zmienił scheduling: %#v", monday.Scheduling)
	}
}

// TestSameDayPairsOneSidedEqualsBothSided — relacja podana raz musi dać ten sam
// profil co zapis obustronny.
func TestSameDayPairsOneSidedEqualsBothSided(t *testing.T) {
	application, personID := newToleranceApplication(t)
	bothSided := parseToleranceProfile(t, application, toleranceJSON(legacyFullAIResponse, personID))

	oneSidedSource := strings.Replace(legacyFullAIResponse,
		`"same_day_pairs": [["program_a", "program_b"], ["program_b", "program_a"]],`,
		`"same_day_pairs": [["program_a", "program_b"]],`, 1)
	if oneSidedSource == legacyFullAIResponse {
		t.Fatal("nie udało się przygotować wariantu jednostronnego")
	}
	oneSided := parseToleranceProfile(t, application, toleranceJSON(oneSidedSource, personID))

	if !reflect.DeepEqual(bothSided, oneSided) {
		t.Fatalf("jednostronne same_day_pairs dały inny profil:\njedno: %#v\noba:   %#v", oneSided, bothSided)
	}
}

// TestSameDayPairsClosureIsIdempotent — powtórzenia i duplikaty par nie
// duplikują wpisów w SameDayWith.
func TestSameDayPairsClosureIsIdempotent(t *testing.T) {
	application, personID := newToleranceApplication(t)
	noisySource := strings.Replace(legacyFullAIResponse,
		`"same_day_pairs": [["program_a", "program_b"], ["program_b", "program_a"]],`,
		`"same_day_pairs": [["program_a", "program_b"], ["program_b", "program_a"], ["program_a", "program_b"]],`, 1)
	profile := parseToleranceProfile(t, application, toleranceJSON(noisySource, personID))
	if got := profile.Phases[0].Scheduling.SameDayWith; !reflect.DeepEqual(got, []string{"program_b"}) {
		t.Fatalf("domknięcie zduplikowało wpisy: %#v", got)
	}
}

const minimalAIResponse = `{
  "person_id": "%PERSON%",
  "programs": [
    {
      "id": "program_a",
      "name": "Program A",
      "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]
    }
  ],
  "phases": [
    {"name": "Start", "days": 7, "program": "program_a"},
    {"name": "Tydzien", "days": 14, "week": {"Monday": "program_a"}}
  ]
}`

// TestOptionalFieldsFallBackToDefaults — pominięte pola opcjonalne dostają
// wartości domyślne po stronie Go.
func TestOptionalFieldsFallBackToDefaults(t *testing.T) {
	application, personID := newToleranceApplication(t)
	profile := parseToleranceProfile(t, application, toleranceJSON(minimalAIResponse, personID))

	want := SessionScheduling{ProgramID: "program_a", Repetitions: 1, SameDayWith: []string{}}
	if !reflect.DeepEqual(profile.Phases[0].Scheduling, want) {
		t.Fatalf("domyślny scheduling fazy: %#v, oczekiwano %#v", profile.Phases[0].Scheduling, want)
	}
	if profile.Phases[0].IntervalGap != 0 {
		t.Fatalf("brak every_days powinien znaczyć codziennie (gap 0), jest %d", profile.Phases[0].IntervalGap)
	}
	monday := profile.Phases[1].Schedule["Monday"]
	if !reflect.DeepEqual(monday.Scheduling, want) {
		t.Fatalf("domyślny scheduling dnia: %#v, oczekiwano %#v", monday.Scheduling, want)
	}
}

// TestExplicitZeroIsNotConfusedWithMissingField — jawne 0 tam, gdzie jest
// dozwolone, działa; jawne 0 tam, gdzie jest zabronione, jest odrzucane.
func TestExplicitZeroIsNotConfusedWithMissingField(t *testing.T) {
	application, personID := newToleranceApplication(t)

	explicitZeros := `{
  "person_id": "%PERSON%",
  "programs": [{
    "id": "program_a",
    "name": "Program A",
    "steps": [{"frequency_hz": 30000, "duration_seconds": 420}],
    "min_gap_minutes": 0,
    "cooldown_after_minutes": 0
  }],
  "phases": [{"name": "Start", "days": 7, "every_days": 1, "program": "program_a", "break_minutes": 0}]
}`
	profile := parseToleranceProfile(t, application, toleranceJSON(explicitZeros, personID))
	want := SessionScheduling{ProgramID: "program_a", Repetitions: 1, SameDayWith: []string{}}
	if !reflect.DeepEqual(profile.Phases[0].Scheduling, want) {
		t.Fatalf("jawne zera dały inny scheduling: %#v", profile.Phases[0].Scheduling)
	}

	// repeat: 0 jest wartością obecną i poza zakresem 1–12 — musi być odrzucone,
	// a nie po cichu potraktowane jak brak pola.
	explicitZeroRepeat := `{
  "person_id": "%PERSON%",
  "programs": [{"id": "program_a", "name": "Program A", "steps": [{"frequency_hz": 30000, "duration_seconds": 420}]}],
  "phases": [{"name": "Start", "days": 7, "every_days": 1, "program": "program_a", "repeat": 0}]
}`
	if _, _, err := application.parseAIProfileLocked(toleranceJSON(explicitZeroRepeat, personID)); err == nil {
		t.Fatal("repeat: 0 powinno zostać odrzucone jako wartość poza zakresem")
	} else if !strings.Contains(err.Error(), "repeat") {
		t.Fatalf("błąd dla repeat: 0 jest nieczytelny: %v", err)
	}
}

// TestOutOfRangeValuesStillRejected — zakresy chroniące przed realnym błędem
// zostają ostre.
func TestOutOfRangeValuesStillRejected(t *testing.T) {
	application, personID := newToleranceApplication(t)

	tooManySteps := strings.Repeat(`{"frequency_hz": 100, "duration_seconds": 60},`, 13)
	tooManySteps = strings.TrimSuffix(tooManySteps, ",")

	cases := []struct {
		name     string
		body     string
		contains string
	}{
		{
			name: "duration_seconds=0",
			body: `{"person_id":"%PERSON%","programs":[{"id":"program_a","name":"A","steps":[{"frequency_hz":30000,"duration_seconds":0}]}],
			        "phases":[{"name":"Start","days":7,"every_days":1,"program":"program_a"}]}`,
			contains: "czas",
		},
		{
			name: "repeat=13",
			body: `{"person_id":"%PERSON%","programs":[{"id":"program_a","name":"A","steps":[{"frequency_hz":30000,"duration_seconds":420}]}],
			        "phases":[{"name":"Start","days":7,"every_days":1,"program":"program_a","repeat":13}]}`,
			contains: "repeat",
		},
		{
			name: "every_days=0",
			body: `{"person_id":"%PERSON%","programs":[{"id":"program_a","name":"A","steps":[{"frequency_hz":30000,"duration_seconds":420}]}],
			        "phases":[{"name":"Start","days":7,"every_days":0,"program":"program_a"}]}`,
			contains: "every_days",
		},
		{
			name: "13 kroków",
			body: `{"person_id":"%PERSON%","programs":[{"id":"program_a","name":"A","steps":[` + tooManySteps + `]}],
			        "phases":[{"name":"Start","days":7,"every_days":1,"program":"program_a"}]}`,
			contains: "kroków",
		},
		{
			name: "brak duration_seconds",
			body: `{"person_id":"%PERSON%","programs":[{"id":"program_a","name":"A","steps":[{"frequency_hz":30000}]}],
			        "phases":[{"name":"Start","days":7,"every_days":1,"program":"program_a"}]}`,
			contains: "czas",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := application.parseAIProfileLocked(toleranceJSON(testCase.body, personID))
			if err == nil {
				t.Fatal("oczekiwano błędu, parser przyjął dane")
			}
			if !strings.Contains(err.Error(), testCase.contains) {
				t.Fatalf("błąd %q nie zawiera %q", err.Error(), testCase.contains)
			}
		})
	}
}
