package main

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateGitHubOwner = "snakex21"
	updateGitHubRepo  = "Zapper"
)

// Używane wyłącznie do sprzątania starych plików w paczce portable.
// Aktualizator nie może zależeć od tej flagi. Pakiet main nadpisujemy przez
// -X main.appBuildFlavor=portable.
var appBuildFlavor = "development"

type AppUpdateInfo struct {
	CurrentVersion   string `json:"current_version"`
	LatestVersion    string `json:"latest_version"`
	Available        bool   `json:"available"`
	InstallSupported bool   `json:"install_supported"`
	ReleaseURL       string `json:"release_url"`
	PublishedAt      string `json:"published_at"`
	Notes            string `json:"notes"`
	ZipAssetName     string `json:"zip_asset_name"`
}

type AppUpdateInstallResult struct {
	Version    string `json:"version"`
	Restarting bool   `json:"restarting"`
}

type githubRelease struct {
	TagName     string               `json:"tag_name"`
	HTMLURL     string               `json:"html_url"`
	Body        string               `json:"body"`
	PublishedAt string               `json:"published_at"`
	Assets      []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func checkAppUpdate(_ string) (AppUpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	release, err := fetchLatestGitHubRelease(ctx)
	if err != nil {
		return AppUpdateInfo{}, err
	}
	return appUpdateInfoFromRelease(release, appVersion)
}

func appUpdateInfoFromRelease(release githubRelease, currentVersion string) (AppUpdateInfo, error) {
	latest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(release.TagName), "v"))
	if latest == "" {
		return AppUpdateInfo{}, errors.New("GitHub release nie ma poprawnego tagu wersji")
	}
	newer, err := versionNewer(latest, currentVersion)
	if err != nil {
		return AppUpdateInfo{}, fmt.Errorf("nieprawidłowa wersja release %q: %w", release.TagName, err)
	}

	zipName := fmt.Sprintf("Zapper-v%s-Windows-x64.zip", latest)
	shaName := zipName + ".sha256"
	zipURL, shaURL := releaseAssetURLs(release, zipName, shaName)
	assetsReady := zipURL != "" && shaURL != ""

	return AppUpdateInfo{
		CurrentVersion:   currentVersion,
		LatestVersion:    latest,
		Available:        newer,
		InstallSupported: newer && runtime.GOOS == "windows" && assetsReady,
		ReleaseURL:       release.HTMLURL,
		PublishedAt:      release.PublishedAt,
		Notes:            release.Body,
		ZipAssetName:     zipName,
	}, nil
}

func installLatestAppUpdate(appDirectory string) (AppUpdateInstallResult, error) {
	if runtime.GOOS != "windows" {
		return AppUpdateInstallResult{}, errors.New("automatyczna instalacja aktualizacji jest obsługiwana tylko w Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	release, err := fetchLatestGitHubRelease(ctx)
	if err != nil {
		return AppUpdateInstallResult{}, err
	}
	latest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(release.TagName), "v"))
	newer, err := versionNewer(latest, appVersion)
	if err != nil {
		return AppUpdateInstallResult{}, err
	}
	if !newer {
		return AppUpdateInstallResult{}, errors.New("nie ma nowszej wersji do zainstalowania")
	}

	zipName := fmt.Sprintf("Zapper-v%s-Windows-x64.zip", latest)
	shaName := zipName + ".sha256"
	zipURL, shaURL := releaseAssetURLs(release, zipName, shaName)
	if zipURL == "" || shaURL == "" {
		return AppUpdateInstallResult{}, fmt.Errorf("release v%s nie zawiera wymaganych plików %s i %s", latest, zipName, shaName)
	}

	// Aktualizacja jest częścią danych aplikacji portable. Nie używamy
	// systemowego TEMP ani katalogu profilu użytkownika, dzięki czemu cały
	// mechanizm pozostaje przenośny razem z folderem Zappera.
	updateDataRoot := filepath.Join(appDirectory, "data", "update-staging")
	if err := os.MkdirAll(updateDataRoot, 0o755); err != nil {
		return AppUpdateInstallResult{}, fmt.Errorf("nie udało się przygotować lokalnego katalogu aktualizacji: %w", err)
	}
	updateRoot, err := os.MkdirTemp(updateDataRoot, "ZapperUpdate-"+sanitizeVersion(latest)+"-")
	if err != nil {
		return AppUpdateInstallResult{}, fmt.Errorf("nie udało się przygotować katalogu aktualizacji: %w", err)
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			_ = os.RemoveAll(updateRoot)
		}
	}()

	zipPath := filepath.Join(updateRoot, zipName)
	if err := downloadToFile(ctx, zipURL, zipPath); err != nil {
		return AppUpdateInstallResult{}, fmt.Errorf("nie udało się pobrać aktualizacji: %w", err)
	}
	shaText, err := downloadSmallText(ctx, shaURL, 16*1024)
	if err != nil {
		return AppUpdateInstallResult{}, fmt.Errorf("nie udało się pobrać sumy SHA-256: %w", err)
	}
	expectedHash, err := parseSHA256File(shaText, zipName)
	if err != nil {
		return AppUpdateInstallResult{}, err
	}
	actualHash, err := fileSHA256(zipPath)
	if err != nil {
		return AppUpdateInstallResult{}, err
	}
	if !strings.EqualFold(expectedHash, actualHash) {
		return AppUpdateInstallResult{}, fmt.Errorf("suma SHA-256 paczki jest nieprawidłowa: oczekiwano %s, otrzymano %s", expectedHash, actualHash)
	}

	extractRoot := filepath.Join(updateRoot, "payload")
	if err := extractZip(zipPath, extractRoot); err != nil {
		return AppUpdateInstallResult{}, fmt.Errorf("nie udało się rozpakować aktualizacji: %w", err)
	}
	payloadRoot, err := findPortablePayload(extractRoot)
	if err != nil {
		return AppUpdateInstallResult{}, err
	}
	if err := startUpdateInstaller(appDirectory, payloadRoot, updateRoot); err != nil {
		return AppUpdateInstallResult{}, err
	}
	cleanupOnError = false
	return AppUpdateInstallResult{Version: latest, Restarting: true}, nil
}

func fetchLatestGitHubRelease(ctx context.Context) (githubRelease, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", updateGitHubOwner, updateGitHubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Zapper/"+appVersion)
	resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("nie udało się połączyć z GitHubem: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return githubRelease{}, fmt.Errorf("GitHub zwrócił HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2*1024*1024)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("nie udało się odczytać informacji o release: %w", err)
	}
	return release, nil
}

func releaseAssetURLs(release githubRelease, zipName, shaName string) (string, string) {
	var zipURL, shaURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case zipName:
			zipURL = asset.BrowserDownloadURL
		case shaName:
			shaURL = asset.BrowserDownloadURL
		}
	}
	return zipURL, shaURL
}

func versionNewer(candidate, current string) (bool, error) {
	candidateParts, err := parseNumericVersion(candidate)
	if err != nil {
		return false, err
	}
	currentParts, err := parseNumericVersion(current)
	if err != nil {
		return false, err
	}
	for i := 0; i < 3; i++ {
		if candidateParts[i] > currentParts[i] {
			return true, nil
		}
		if candidateParts[i] < currentParts[i] {
			return false, nil
		}
	}
	return false, nil
}

func parseNumericVersion(value string) ([3]int, error) {
	var result [3]int
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "v"))
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("wersja %q nie ma formatu x.y.z", value)
	}
	for index, part := range parts {
		if part == "" {
			return result, fmt.Errorf("wersja %q ma pusty segment", value)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return result, fmt.Errorf("wersja %q zawiera nieprawidłowy segment %q", value, part)
		}
		result[index] = number
	}
	return result, nil
}

func sanitizeVersion(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= '0' && char <= '9') || char == '.' || char == '-' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "unknown"
	}
	return builder.String()
}

func downloadToFile(ctx context.Context, sourceURL, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Zapper/"+appVersion)
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	return file.Sync()
}

func downloadSmallText(ctx context.Context, sourceURL string, limit int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Zapper/"+appVersion)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(body)) > limit {
		return "", errors.New("plik sumy kontrolnej jest zbyt duży")
	}
	return string(body), nil
}

func parseSHA256File(content, expectedFileName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name != expectedFileName {
			continue
		}
		hash := strings.ToLower(fields[0])
		decoded, err := hex.DecodeString(hash)
		if err != nil || len(decoded) != sha256.Size {
			return "", errors.New("plik SHA-256 zawiera nieprawidłową sumę")
		}
		return hash, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("plik SHA-256 nie zawiera wpisu dla %s", expectedFileName)
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractZip(source, destination string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	cleanRoot := filepath.Clean(destination) + string(os.PathSeparator)
	for _, entry := range archive.File {
		entryPath := filepath.Join(destination, filepath.FromSlash(entry.Name))
		cleanPath := filepath.Clean(entryPath)
		if !strings.HasPrefix(cleanPath+string(os.PathSeparator), cleanRoot) {
			return fmt.Errorf("niebezpieczna ścieżka w ZIP: %s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
			return err
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		mode := entry.Mode()
		if mode == 0 {
			mode = 0o644
		}
		writer, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			reader.Close()
			return err
		}
		_, copyErr := io.Copy(writer, reader)
		closeErr := writer.Close()
		reader.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func findPortablePayload(extractRoot string) (string, error) {
	candidates := []string{extractRoot}
	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(extractRoot, entry.Name()))
		}
	}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(candidate, "Zapper.exe")) &&
			directoryExists(filepath.Join(candidate, "locales")) &&
			directoryExists(filepath.Join(candidate, "firmware", "localized")) {
			return candidate, nil
		}
	}
	return "", errors.New("paczka aktualizacji nie zawiera poprawnej wersji portable Zapper")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
