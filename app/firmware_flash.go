package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const latestFirmwareVersion = "5.1.0"

var firmwareLCDEnglishFallback = map[string]struct{}{
	"ru": {}, "uk": {}, "bg": {}, "el": {}, "ar": {}, "he": {}, "hi": {}, "zh": {}, "ja": {}, "ko": {},
}

type FirmwareFlashInfo struct {
	LatestVersion   string `json:"latest_version"`
	Language        string `json:"language"`
	LCDLanguage     string `json:"lcd_language"`
	EnglishFallback bool   `json:"english_fallback"`
	VariantName     string `json:"variant_name"`
	SketchAvailable bool   `json:"sketch_available"`
	SketchPath      string `json:"sketch_path,omitempty"`
	ToolAvailable   bool   `json:"tool_available"`
	ToolPath        string `json:"tool_path,omitempty"`
}

type FirmwareFlashRequest struct {
	Port          string `json:"port"`
	Language      string `json:"language"`
	OldBootloader bool   `json:"old_bootloader"`
}

type FirmwareFlashResult struct {
	OK            bool   `json:"ok"`
	Version       string `json:"version"`
	Language      string `json:"language"`
	LCDLanguage   string `json:"lcd_language"`
	OldBootloader bool   `json:"old_bootloader"`
	Tool          string `json:"tool"`
	Output        string `json:"output"`
}

func firmwareLanguageDetails(language string) (string, string, bool) {
	language = normalizeLanguageCode(language)
	if language == "" {
		language = "en"
	}
	_, fallback := firmwareLCDEnglishFallback[language]
	lcdLanguage := language
	if fallback {
		lcdLanguage = "en"
	}
	return language, lcdLanguage, fallback
}

func locateFirmwareSketch(appDirectory, language string) string {
	language, _, _ = firmwareLanguageDetails(language)
	folder := "zapper_v5_" + language
	file := folder + ".ino"
	candidates := []string{
		filepath.Join(appDirectory, "firmware", "localized", folder, file),
		filepath.Join(appDirectory, "build", "generated", "firmware", folder, file),
		filepath.Join(appDirectory, "firmware", folder, file),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func locateArduinoCLI(appDirectory string) string {
	binary := "arduino-cli"
	if runtime.GOOS == "windows" {
		binary = "arduino-cli.exe"
	}
	candidates := []string{
		filepath.Join(appDirectory, "tools", "arduino-cli", binary),
		filepath.Join(appDirectory, "tools", binary),
		filepath.Join(appDirectory, binary),
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates, filepath.Join(local, "Programs", "Arduino IDE", "resources", "app", "lib", "backend", "resources", "arduino-cli.exe"))
		}
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			candidates = append(candidates, filepath.Join(programFiles, "Arduino IDE", "resources", "app", "lib", "backend", "resources", "arduino-cli.exe"))
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	if path, err := exec.LookPath(binary); err == nil {
		return path
	}
	return ""
}

func firmwareFlashInfo(appDirectory, language string) FirmwareFlashInfo {
	language, lcdLanguage, fallback := firmwareLanguageDetails(language)
	sketch := locateFirmwareSketch(appDirectory, language)
	tool := locateArduinoCLI(appDirectory)
	return FirmwareFlashInfo{
		LatestVersion:   latestFirmwareVersion,
		Language:        language,
		LCDLanguage:     lcdLanguage,
		EnglishFallback: fallback,
		VariantName:     "zapper_v5_" + language,
		SketchAvailable: sketch != "",
		SketchPath:      sketch,
		ToolAvailable:   tool != "",
		ToolPath:        tool,
	}
}

var windowsCOMPattern = regexp.MustCompile(`(?i)^COM[1-9][0-9]*$`)

func validateFlashPort(port string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return errors.New("nie wybrano portu urządzenia")
	}
	if runtime.GOOS == "windows" && !windowsCOMPattern.MatchString(port) {
		return fmt.Errorf("nieprawidłowy port COM: %s", port)
	}
	return nil
}

func flashFirmware(appDirectory string, request FirmwareFlashRequest) (FirmwareFlashResult, error) {
	request.Port = strings.TrimSpace(request.Port)
	if err := validateFlashPort(request.Port); err != nil {
		return FirmwareFlashResult{}, err
	}
	language, lcdLanguage, _ := firmwareLanguageDetails(request.Language)
	sketch := locateFirmwareSketch(appDirectory, language)
	if sketch == "" {
		return FirmwareFlashResult{}, fmt.Errorf("nie znaleziono wariantu firmware dla języka %s", language)
	}
	tool := locateArduinoCLI(appDirectory)
	if tool == "" {
		return FirmwareFlashResult{}, errors.New("nie znaleziono arduino-cli; umieść arduino-cli w folderze tools/arduino-cli albo zainstaluj Arduino IDE/Arduino CLI")
	}

	fqbn := "arduino:avr:nano:cpu=atmega328"
	if request.OldBootloader {
		fqbn = "arduino:avr:nano:cpu=atmega328old"
	}

	buildRoot := filepath.Join(appDirectory, "data", "firmware-build")
	if err := os.MkdirAll(buildRoot, 0o755); err != nil {
		return FirmwareFlashResult{}, fmt.Errorf("nie udało się przygotować lokalnego katalogu kompilacji firmware: %w", err)
	}
	buildDir, err := os.MkdirTemp(buildRoot, "zapper-firmware-*")
	if err != nil {
		return FirmwareFlashResult{}, err
	}
	defer os.RemoveAll(buildDir)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	args := []string{
		"compile",
		"--fqbn", fqbn,
		"--build-path", buildDir,
		"--upload",
		"--port", request.Port,
		filepath.Dir(sketch),
	}
	command := exec.CommandContext(ctx, tool, args...)
	output, runErr := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		return FirmwareFlashResult{}, errors.New("wgrywanie firmware przekroczyło limit czasu")
	}
	if runErr != nil {
		if text == "" {
			text = runErr.Error()
		}
		return FirmwareFlashResult{
			OK:            false,
			Version:       latestFirmwareVersion,
			Language:      language,
			LCDLanguage:   lcdLanguage,
			OldBootloader: request.OldBootloader,
			Tool:          tool,
			Output:        text,
		}, fmt.Errorf("arduino-cli nie wgrało firmware: %s", text)
	}
	return FirmwareFlashResult{
		OK:            true,
		Version:       latestFirmwareVersion,
		Language:      language,
		LCDLanguage:   lcdLanguage,
		OldBootloader: request.OldBootloader,
		Tool:          tool,
		Output:        text,
	}, nil
}
