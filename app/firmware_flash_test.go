package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFirmwareLanguageFallback(t *testing.T) {
	language, lcd, fallback := firmwareLanguageDetails("ru-RU")
	if language != "ru" || lcd != "en" || !fallback {
		t.Fatalf("nieprawidłowy fallback ru: language=%q lcd=%q fallback=%v", language, lcd, fallback)
	}
	language, lcd, fallback = firmwareLanguageDetails("pl-PL")
	if language != "pl" || lcd != "pl" || fallback {
		t.Fatalf("nieprawidłowy wariant pl: language=%q lcd=%q fallback=%v", language, lcd, fallback)
	}
}

func TestLocateLocalizedFirmwareSketch(t *testing.T) {
	directory := t.TempDir()
	folder := filepath.Join(directory, "firmware", "localized", "zapper_v5_de")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, "zapper_v5_de.ino")
	if err := os.WriteFile(path, []byte("// test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := locateFirmwareSketch(directory, "de-DE"); got != path {
		t.Fatalf("oczekiwano %q, otrzymano %q", path, got)
	}
}

func TestLocateBundledArduinoCLI(t *testing.T) {
	directory := t.TempDir()
	binary := "arduino-cli"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	path := filepath.Join(directory, "tools", "arduino-cli", binary)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := locateArduinoCLI(directory); got != path {
		t.Fatalf("oczekiwano bundled arduino-cli %q, otrzymano %q", path, got)
	}
}
