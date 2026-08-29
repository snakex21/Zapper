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
