//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparePortableInstallRemovesObsoletePackageFiles(t *testing.T) {
	directory := t.TempDir()
	obsolete := []string{
		".zapper-portable",
		"LICENSE",
		"README_PORTABLE.txt",
		"Zapper.ico",
		filepath.Join("firmware", "languages.json"),
		filepath.Join("firmware", "LANGUAGES.md"),
		filepath.Join("firmware", "archive", "old.ino"),
		filepath.Join("firmware", "zapper_v5", "zapper_v5.ino"),
	}
	for _, relative := range obsolete {
		path := filepath.Join(directory, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("obsolete"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	previousFlavor := appBuildFlavor
	appBuildFlavor = "portable"
	t.Cleanup(func() { appBuildFlavor = previousFlavor })
	preparePortableInstall(directory)

	for _, relative := range obsolete {
		if _, err := os.Stat(filepath.Join(directory, relative)); !os.IsNotExist(err) {
			t.Fatalf("zbędny element paczki nadal istnieje: %s", relative)
		}
	}
}

func TestPreparePortableInstallDoesNotCleanDevelopmentDirectory(t *testing.T) {
	directory := t.TempDir()
	license := filepath.Join(directory, "LICENSE")
	if err := os.WriteFile(license, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	previousFlavor := appBuildFlavor
	appBuildFlavor = "development"
	t.Cleanup(func() { appBuildFlavor = previousFlavor })
	preparePortableInstall(directory)
	if _, err := os.Stat(license); err != nil {
		t.Fatalf("build developerski usunął plik repozytorium: %v", err)
	}
}
