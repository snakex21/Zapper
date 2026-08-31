//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPreparePortableInstallHidesMarkerAndRemovesObsoletePackageFiles(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, portableMarker)
	if err := os.WriteFile(marker, []byte("portable"), 0o644); err != nil {
		t.Fatal(err)
	}
	obsolete := []string{
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

	preparePortableInstall(directory)

	markerUTF16, err := syscall.UTF16PtrFromString(marker)
	if err != nil {
		t.Fatal(err)
	}
	attributes, err := syscall.GetFileAttributes(markerUTF16)
	if err != nil {
		t.Fatal(err)
	}
	if attributes&syscall.FILE_ATTRIBUTE_HIDDEN == 0 {
		t.Fatal("znacznik portable nie został ukryty")
	}
	for _, relative := range obsolete {
		if _, err := os.Stat(filepath.Join(directory, relative)); !os.IsNotExist(err) {
			t.Fatalf("zbędny element paczki nadal istnieje: %s", relative)
		}
	}
}
