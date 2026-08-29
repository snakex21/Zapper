package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type AppSettings struct {
	Language       string `json:"language,omitempty"`
	LanguageSource string `json:"language_source,omitempty"`
}

var supportedLanguages = map[string]struct{}{
	"en": {}, "pl": {}, "de": {}, "fr": {}, "es": {}, "it": {}, "pt": {}, "nl": {}, "sv": {}, "no": {},
	"da": {}, "fi": {}, "cs": {}, "sk": {}, "hu": {}, "ro": {}, "tr": {}, "id": {}, "ms": {}, "vi": {},
	"ru": {}, "uk": {}, "bg": {}, "el": {}, "ar": {}, "he": {}, "hi": {}, "zh": {}, "ja": {}, "ko": {},
}

func normalizeLanguageCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	if value == "" {
		return ""
	}
	if dash := strings.IndexByte(value, '-'); dash >= 0 {
		value = value[:dash]
	}
	if _, ok := supportedLanguages[value]; !ok {
		return ""
	}
	return value
}

func appSettingsPath(appDirectory string) string {
	return filepath.Join(appDirectory, "data", "settings.json")
}

func loadAppSettings(appDirectory string) (AppSettings, error) {
	path := appSettingsPath(appDirectory)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AppSettings{}, nil
	}
	if err != nil {
		return AppSettings{}, err
	}
	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return AppSettings{}, err
	}
	settings.Language = normalizeLanguageCode(settings.Language)
	if settings.Language == "" {
		settings.LanguageSource = ""
	}
	return settings, nil
}

func saveAppSettings(appDirectory string, settings AppSettings) (AppSettings, error) {
	settings.Language = normalizeLanguageCode(settings.Language)
	if settings.Language == "" {
		return AppSettings{}, errors.New("nieobsługiwany język aplikacji")
	}
	if settings.LanguageSource != "manual" {
		settings.LanguageSource = "auto"
	}
	if err := os.MkdirAll(filepath.Join(appDirectory, "data"), 0o755); err != nil {
		return AppSettings{}, err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return AppSettings{}, err
	}
	if err := os.WriteFile(appSettingsPath(appDirectory), data, 0o644); err != nil {
		return AppSettings{}, err
	}
	return settings, nil
}
