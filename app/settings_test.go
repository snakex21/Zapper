package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeLanguageCode(t *testing.T) {
	cases := map[string]string{
		"pl-PL": "pl",
		"PT_BR": "pt",
		"zh-CN": "zh",
		"RU":    "ru",
		"xx-YY": "",
		"":      "",
	}
	for input, want := range cases {
		if got := normalizeLanguageCode(input); got != want {
			t.Fatalf("normalizeLanguageCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAppSettingsRoundTrip(t *testing.T) {
	directory := t.TempDir()
	if got, err := loadAppSettings(directory); err != nil || got.Language != "" {
		t.Fatalf("empty settings: %#v, %v", got, err)
	}

	saved, err := saveAppSettings(directory, AppSettings{Language: "pl-PL", LanguageSource: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Language != "pl" || saved.LanguageSource != "auto" {
		t.Fatalf("unexpected saved settings: %#v", saved)
	}
	if _, err := os.Stat(filepath.Join(directory, "data", "settings.json")); err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	loaded, err := loadAppSettings(directory)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != saved {
		t.Fatalf("round trip mismatch: got %#v want %#v", loaded, saved)
	}
}

func TestAppSettingsRejectUnsupportedLanguage(t *testing.T) {
	if _, err := saveAppSettings(t.TempDir(), AppSettings{Language: "xx"}); err == nil {
		t.Fatal("unsupported language should be rejected")
	}
}

// Notatnik Windows zapisuje "UTF-8 z BOM"; taki plik nie może blokować startu
// komunikatem "invalid character 'ï'".
func TestJSONDataFilesAcceptUTF8BOM(t *testing.T) {
	directory := t.TempDir()
	dataDirectory := filepath.Join(directory, "data")
	if err := os.MkdirAll(dataDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	bom := "\xEF\xBB\xBF"
	write := func(name, content string) string {
		path := filepath.Join(dataDirectory, name)
		if err := os.WriteFile(path, []byte(bom+content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	write("settings.json", `{"language":"de","language_source":"manual"}`)
	settings, err := loadAppSettings(directory)
	if err != nil {
		t.Fatalf("settings.json with BOM: %v", err)
	}
	if settings.Language != "de" || settings.LanguageSource != "manual" {
		t.Fatalf("unexpected settings: %#v", settings)
	}

	devicePath := write("device.json", `{"port":"COM7"}`)
	var preferences DevicePreferences
	if err := loadJSON(devicePath, &preferences); err != nil {
		t.Fatalf("loadJSON with BOM: %v", err)
	}
	if preferences.Port != "COM7" {
		t.Fatalf("unexpected device preferences: %#v", preferences)
	}

	write("window.json", `{"width":1500,"height":900,"maximized":true}`)
	if state := loadWindowState(directory); state.Width != 1500 || state.Height != 900 {
		t.Fatalf("window.json with BOM ignored: %#v", state)
	}

	if got := string(stripUTF8BOM([]byte(bom + "{}"))); got != "{}" {
		t.Fatalf("stripUTF8BOM = %q", got)
	}
	if got := string(stripUTF8BOM([]byte("{}"))); got != "{}" {
		t.Fatalf("stripUTF8BOM without BOM = %q", got)
	}
}
