package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func backendEdgeFixture(t *testing.T) (*Application, string, string) {
	t.Helper()
	dir := t.TempDir()
	a, e := NewApplication(dir)
	if e != nil {
		t.Fatal(e)
	}
	a.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local) }
	p, e := a.AddPerson("Test", "")
	if e != nil {
		t.Fatal(e)
	}
	raw := `{"person_id":"` + p.PersonID + `","phases":[{"name":"Start","days":7,"frequency_hz":30000,"duration_seconds":420}]}`
	if _, e = a.ApplyAIProfile(raw); e != nil {
		t.Fatal(e)
	}
	return a, dir, raw
}

func TestPartialRunSurvivesRestart(t *testing.T) {
	a, dir, _ := backendEdgeFixture(t)
	s := a.Snapshot().Today[0].SessionID
	_, e := a.SavePausedSession(DevicePauseState{SessionID: s, RemainingSteps: []DeviceStep{{FrequencyMilliHz: 30000000, DurationSeconds: 216}}, RemainingSeconds: 216}, true)
	if e != nil {
		t.Fatal(e)
	}
	b, e := NewApplication(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("partial runs in memory=%d after restart=%d; pause survives=%t", len(a.progress.PartialRuns), len(b.progress.PartialRuns), len(b.progress.PausedSessions) > 0)
	if !reflect.DeepEqual(b.progress.PartialRuns, a.progress.PartialRuns) {
		t.Error("partial run changed or lost after restart")
	}
}

func TestStoppingArmedSessionRecordsZeroElapsed(t *testing.T) {
	a, _, _ := backendEdgeFixture(t)
	plan := a.Snapshot().Today[0]
	d := &DeviceManager{port: &fakeSerialConnection{}, status: DeviceStatus{Connected: true, Ready: true, State: "idle"}}
	if _, e := d.Start(plan.SessionID, plan.ProfileName, plan.DeviceSteps); e != nil {
		t.Fatal(e)
	}
	d.handleLine("ARMED 1", 0)
	pause, _, e := d.Pause()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.SavePausedSession(pause, true); e != nil {
		t.Fatal(e)
	}
	r := a.progress.PartialRuns[0]
	t.Logf("Start -> ARMED -> Pause -> SavePausedSession(recordPartial=true) reports done=%d total=%d remaining=%d", r.DoneSeconds, r.TotalSeconds, r.RemainingSeconds)
	if r.DoneSeconds != 0 {
		t.Error("stop before hardware confirmation records full session duration as done")
	}
}

func TestSingleArrayUnknownFieldsMatchObject(t *testing.T) {
	a, _, raw := backendEdgeFixture(t)
	raw = strings.Replace(raw, `"days":7`, `"days":7,"unexpected":true`, 1)
	object, e := a.PreviewAIProfile(raw)
	if e != nil {
		t.Fatal(e)
	}
	array, e := a.PreviewAIProfile("[" + raw + "]")
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("object warnings=%v single-array warnings=%v", object.Persons[0].UnknownFields, array.Persons[0].UnknownFields)
	if len(object.Persons[0].UnknownFields) != 1 || !reflect.DeepEqual(array.Persons[0].UnknownFields, object.Persons[0].UnknownFields) {
		t.Error("single array silently drops unknown-field warnings")
	}
}

func TestPartialRunProfileAndPersonFiltering(t *testing.T) {
	source := Progress{PartialRuns: []PartialRun{
		{SessionID: "keep", ProfileID: "p", PersonID: "person", RunID: "current"},
		{SessionID: "old", ProfileID: "p", PersonID: "person", RunID: "old"},
		{SessionID: "other", ProfileID: "other", PersonID: "other", RunID: "current"},
	}}
	profile := Profile{ID: "p", RunID: "current"}
	filtered := progressForProfile(source, profile)
	if len(filtered.PartialRuns) != 1 || filtered.PartialRuns[0].SessionID != "keep" {
		t.Fatalf("profile filtering: %#v", filtered.PartialRuns)
	}
	without := progressWithoutProfile(source, profile)
	if len(without.PartialRuns) != 2 || without.PartialRuns[0].SessionID != "old" || without.PartialRuns[1].SessionID != "other" {
		t.Fatalf("profile removal: %#v", without.PartialRuns)
	}
	without = progressWithoutPerson(source, "person")
	if len(without.PartialRuns) != 1 || without.PartialRuns[0].SessionID != "other" {
		t.Fatalf("person removal: %#v", without.PartialRuns)
	}
	profile.RunID = ""
	if len(progressForProfile(source, profile).PartialRuns) != 2 || len(progressWithoutProfile(source, profile).PartialRuns) != 1 {
		t.Fatal("legacy profile must match all its runs")
	}
	// Filtering must not retain the source slice's backing array.
	filtered.PartialRuns[0].DoneSeconds = 99
	if source.PartialRuns[0].DoneSeconds != 0 {
		t.Fatal("filtered records alias source")
	}
}

func TestPartialRunsPersistSeparatelyAndArchive(t *testing.T) {
	a, dir, _ := backendEdgeFixture(t)
	first := a.config.Profiles[0]
	secondPerson, err := a.AddPerson("Other", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyAIProfile(`{"person_id":"` + secondPerson.PersonID + `","phases":[{"name":"Other","days":7,"frequency_hz":1000,"duration_seconds":300}]}`); err != nil {
		t.Fatal(err)
	}
	second := a.config.Profiles[1]
	a.progress.PartialRuns = []PartialRun{
		{SessionID: "first", ProfileID: first.ID, PersonID: first.PersonID, RunID: first.RunID, DoneSeconds: 12},
		{SessionID: "second", ProfileID: second.ID, PersonID: second.PersonID, RunID: second.RunID, DoneSeconds: 34},
	}
	if err = saveActiveProgress(a.progressPath, a.config, a.progress); err != nil {
		t.Fatal(err)
	}
	b, err := NewApplication(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.progress.PartialRuns) != 2 {
		t.Fatalf("restart lost or duplicated records: %#v", b.progress.PartialRuns)
	}
	b.now = a.now
	if _, err = b.FinishProfile(first.ID); err != nil {
		t.Fatal(err)
	}
	if len(b.progress.PartialRuns) != 1 || b.progress.PartialRuns[0].SessionID != "second" {
		t.Fatalf("archiving removed another profile's partial history: %#v", b.progress.PartialRuns)
	}
	c, err := NewApplication(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.progress.PartialRuns) != 1 || c.progress.PartialRuns[0].SessionID != "second" {
		t.Fatalf("remaining active history: %#v", c.progress.PartialRuns)
	}
	if len(c.archive.Profiles) != 1 || c.archive.Profiles[0].Progress == nil || len(c.archive.Profiles[0].Progress.PartialRuns) != 1 || c.archive.Profiles[0].Progress.PartialRuns[0].SessionID != "first" {
		t.Fatalf("archive lost partial history: %#v", c.archive.Profiles)
	}
}

func TestFencedSingleArrayUnknownFields(t *testing.T) {
	a, _, raw := backendEdgeFixture(t)
	raw = strings.Replace(raw, `"days":7`, `"days":7,"unexpected":true`, 1)
	preview, err := a.PreviewAIProfile("```json\n [" + raw + "] \n```")
	if err != nil {
		t.Fatal(err)
	}
	if got := preview.Persons[0].UnknownFields; len(got) != 1 || got[0] != "phases[0].unexpected" {
		t.Fatalf("fenced array warnings: %#v", got)
	}
}
