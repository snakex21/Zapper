package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestReleaseAssetsEnableAutomaticInstallWithoutBuildFlavor(t *testing.T) {
	release := githubRelease{TagName: "v10.4.0", HTMLURL: "https://example.test/release"}
	release.Assets = append(release.Assets,
		githubReleaseAsset{Name: "Zapper-v10.4.0-Windows-x64.zip", BrowserDownloadURL: "https://example.test/Zapper.zip"},
		githubReleaseAsset{Name: "Zapper-v10.4.0-Windows-x64.zip.sha256", BrowserDownloadURL: "https://example.test/Zapper.zip.sha256"},
	)

	info, err := appUpdateInfoFromRelease(release, "10.3.5")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available {
		t.Fatal("wersja 10.4.0 powinna być dostępna dla 10.3.5")
	}
	if info.InstallSupported != (runtime.GOOS == "windows") {
		t.Fatal("kompletne wydanie Windows musi udostępniać automatyczną instalację bez dodatkowej flagi buildu")
	}
}

func TestUpdatePackageDownloadChecksumAndPayloadDiscovery(t *testing.T) {
	packageName := "Zapper-v10.4.0-Windows-x64.zip"
	serverRoot := t.TempDir()
	zipPath := filepath.Join(serverRoot, packageName)
	createTestUpdateZIP(t, zipPath)
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	hashBytes := sha256.Sum256(zipBytes)
	hash := hex.EncodeToString(hashBytes[:])

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/" + packageName:
			response.Header().Set("Content-Type", "application/zip")
			_, _ = response.Write(zipBytes)
		case "/" + packageName + ".sha256":
			_, _ = fmt.Fprintf(response, "%s  %s\n", hash, packageName)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	downloadRoot := t.TempDir()
	downloadedZIP := filepath.Join(downloadRoot, packageName)
	if err := downloadToFile(context.Background(), server.URL+"/"+packageName, downloadedZIP); err != nil {
		t.Fatalf("pobieranie ZIP: %v", err)
	}
	shaText, err := downloadSmallText(context.Background(), server.URL+"/"+packageName+".sha256", 16*1024)
	if err != nil {
		t.Fatalf("pobieranie SHA-256: %v", err)
	}
	expectedHash, err := parseSHA256File(shaText, packageName)
	if err != nil {
		t.Fatalf("odczyt SHA-256: %v", err)
	}
	actualHash, err := fileSHA256(downloadedZIP)
	if err != nil {
		t.Fatalf("obliczanie SHA-256: %v", err)
	}
	if !strings.EqualFold(expectedHash, actualHash) {
		t.Fatalf("SHA-256 pobranej paczki = %s, oczekiwano %s", actualHash, expectedHash)
	}

	extractRoot := filepath.Join(downloadRoot, "payload")
	if err := extractZip(downloadedZIP, extractRoot); err != nil {
		t.Fatalf("rozpakowanie ZIP: %v", err)
	}
	payloadRoot, err := findPortablePayload(extractRoot)
	if err != nil {
		t.Fatalf("wykrycie paczki portable: %v", err)
	}
	if !fileExists(filepath.Join(payloadRoot, "Zapper.exe")) {
		t.Fatal("wykryta paczka nie zawiera Zapper.exe")
	}
}

func createTestUpdateZIP(t *testing.T, destination string) {
	t.Helper()
	file, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for _, name := range []string{
		"Zapper-v10.4.0-Windows-x64/Zapper.exe",
		"Zapper-v10.4.0-Windows-x64/locales/ui.pl.json",
		"Zapper-v10.4.0-Windows-x64/firmware/localized/pl/zapper_v5_pl.ino",
	} {
		writer, createErr := archive.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := writer.Write([]byte("test")); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFindPortablePayloadUsesRequiredStructureWithoutMarkerFile(t *testing.T) {
	extractRoot := t.TempDir()
	payload := filepath.Join(extractRoot, "Zapper-v10.3.9-Windows-x64")
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
