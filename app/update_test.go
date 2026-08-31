package main

import (
	"os"
	"path/filepath"
	"testing"
)

type dispatchingWindowStub struct {
	dispatched func()
	terminated bool
}

func (window *dispatchingWindowStub) Dispatch(callback func()) {
	window.dispatched = callback
}

func (window *dispatchingWindowStub) Terminate() {
	window.terminated = true
}

func TestTerminateWindowIsDispatchedToUIThread(t *testing.T) {
	window := &dispatchingWindowStub{}
	terminateWindowOnUIThread(window)
	if window.terminated {
		t.Fatal("Terminate zostało wywołane na wątku wywołującym zamiast przez Dispatch")
	}
	if window.dispatched == nil {
		t.Fatal("nie przekazano zamknięcia okna do Dispatch")
	}
	window.dispatched()
	if !window.terminated {
		t.Fatal("callback przekazany do Dispatch nie zamyka okna")
	}
}

func TestVersionNewer(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"10.3.1", "10.3.0", true},
		{"10.4.0", "10.3.9", true},
		{"11.0.0", "10.99.99", true},
		{"10.3.0", "10.3.0", false},
		{"10.2.9", "10.3.0", false},
		{"v10.3.1", "10.3.0", true},
	}
	for _, test := range tests {
		got, err := versionNewer(test.candidate, test.current)
		if err != nil {
			t.Fatalf("versionNewer(%q, %q): %v", test.candidate, test.current, err)
		}
		if got != test.want {
			t.Fatalf("versionNewer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestParseSHA256File(t *testing.T) {
	name := "Zapper-v10.4.0-Windows-x64.zip"
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := parseSHA256File(hash+"  "+name+"\n", name)
	if err != nil {
		t.Fatal(err)
	}
	if got != hash {
		t.Fatalf("got %q, want %q", got, hash)
	}
}

func TestFindPortablePayloadUsesRequiredStructureWithoutMarkerFile(t *testing.T) {
	extractRoot := t.TempDir()
	payload := filepath.Join(extractRoot, "Zapper-v10.3.6-Windows-x64")
	for _, directory := range []string{
		filepath.Join(payload, "locales"),
		filepath.Join(payload, "firmware", "localized"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(payload, "Zapper.exe"), []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findPortablePayload(extractRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != payload {
		t.Fatalf("findPortablePayload() = %q, want %q", got, payload)
	}
	if fileExists(filepath.Join(payload, ".zapper-portable")) {
		t.Fatal("testowa paczka nie powinna zawierać znacznika portable")
	}
}
