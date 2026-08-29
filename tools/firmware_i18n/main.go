package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	beginMarker = "// BEGIN GENERATED LANGUAGE PACK"
	endMarker   = "// END GENERATED LANGUAGE PACK"
)

type stringsPack struct {
	ProgramFmt   string `json:"program_fmt"`
	Manual       string `json:"manual"`
	StepTimeFmt  string `json:"step_time_fmt"`
	TimeFmt      string `json:"time_fmt"`
	StartMenu    string `json:"start_menu"`
	USBReadyFmt  string `json:"usb_ready_fmt"`
	StartCancel  string `json:"start_cancel"`
	Stop         string `json:"stop"`
	SessionDone  string `json:"session_done"`
	ClickMenu    string `json:"click_menu"`
}

type language struct {
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	LCDMode  string       `json:"lcd_mode"`
	Fallback string       `json:"fallback"`
	Reason   string       `json:"reason"`
	Strings  *stringsPack `json:"strings"`
}

type manifest struct {
	FirmwareVersion string     `json:"firmware_version"`
	Languages       []language `json:"languages"`
}

func main() {
	root, err := os.Getwd()
	must(err)

	manifestPath := filepath.Join(root, "firmware", "languages.json")
	sourcePath := filepath.Join(root, "firmware", "zapper_v5", "zapper_v5.ino")
	outputRoot := filepath.Join(root, "build", "generated", "firmware")

	var cfg manifest
	data, err := os.ReadFile(manifestPath)
	must(err)
	must(json.Unmarshal(data, &cfg))
	if cfg.FirmwareVersion == "" || len(cfg.Languages) == 0 {
		fatalf("firmware/languages.json is empty or incomplete")
	}

	sourceBytes, err := os.ReadFile(sourcePath)
	must(err)
	source := string(sourceBytes)
	begin := strings.Index(source, beginMarker)
	end := strings.Index(source, endMarker)
	if begin < 0 || end < 0 || end <= begin {
		fatalf("language pack markers were not found in %s", sourcePath)
	}
	end += len(endMarker)

	byCode := make(map[string]language, len(cfg.Languages))
	for _, item := range cfg.Languages {
		if item.Code == "" || item.Name == "" {
			fatalf("language entry without code/name")
		}
		if _, exists := byCode[item.Code]; exists {
			fatalf("duplicate language code %q", item.Code)
		}
		byCode[item.Code] = item
	}

	must(os.RemoveAll(outputRoot))
	must(os.MkdirAll(outputRoot, 0o755))

	var summary []string
	summary = append(summary,
		"ZAPPER FIRMWARE LANGUAGE BUILDS",
		"Firmware: "+cfg.FirmwareVersion,
		"",
		"Each folder is a separate Arduino Nano sketch and contains only one LCD language.",
		"Fallback builds intentionally use the English LCD UI when the standard LCD1602/HD44780 character ROM cannot safely display the requested script.",
		"",
	)

	for _, item := range cfg.Languages {
		pack, actualCode, fallbackNote := resolvePack(item, byCode)
		validatePack(item.Code, pack)

		languageBlock := renderLanguageBlock(cfg.FirmwareVersion, actualCode, pack)
		localized := source[:begin] + languageBlock + source[end:]

		sketchName := "zapper_v5_" + item.Code
		directory := filepath.Join(outputRoot, sketchName)
		must(os.MkdirAll(directory, 0o755))
		must(os.WriteFile(filepath.Join(directory, sketchName+".ino"), []byte(localized), 0o644))

		readme := []string{
			"Zapper firmware " + cfg.FirmwareVersion,
			"Requested language: " + item.Name + " (" + item.Code + ")",
		}
		if fallbackNote == "" {
			readme = append(readme, "LCD language: "+item.Name)
		} else {
			readme = append(readme, "LCD language: English fallback", "Reason: "+fallbackNote)
		}
		readme = append(readme, "", "Open "+sketchName+".ino in Arduino IDE and flash it to Arduino Nano.")
		must(os.WriteFile(filepath.Join(directory, "README.txt"), []byte(strings.Join(readme, "\r\n")+"\r\n"), 0o644))

		status := "LCD native/ASCII"
		if fallbackNote != "" {
			status = "English fallback"
		}
		summary = append(summary, fmt.Sprintf("%-4s %-20s %s", item.Code, item.Name, status))
	}

	must(os.WriteFile(filepath.Join(outputRoot, "LANGUAGES.txt"), []byte(strings.Join(summary, "\r\n")+"\r\n"), 0o644))
	fmt.Printf("Generated %d firmware language builds in %s\n", len(cfg.Languages), outputRoot)
}

func resolvePack(item language, byCode map[string]language) (stringsPack, string, string) {
	if item.Strings != nil {
		return *item.Strings, item.Code, ""
	}
	if item.Fallback == "" {
		fatalf("language %q has neither strings nor fallback", item.Code)
	}
	fallback, ok := byCode[item.Fallback]
	if !ok || fallback.Strings == nil {
		fatalf("language %q points to invalid fallback %q", item.Code, item.Fallback)
	}
	return *fallback.Strings, item.Code + "-" + item.Fallback, item.Reason
}

func validatePack(code string, pack stringsPack) {
	values := map[string]string{
		"program_fmt": pack.ProgramFmt, "manual": pack.Manual, "step_time_fmt": pack.StepTimeFmt,
		"time_fmt": pack.TimeFmt, "start_menu": pack.StartMenu, "usb_ready_fmt": pack.USBReadyFmt,
		"start_cancel": pack.StartCancel, "stop": pack.Stop, "session_done": pack.SessionDone, "click_menu": pack.ClickMenu,
	}
	keys := make([]string, 0, len(values))
	for key := range values { keys = append(keys, key) }
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		if value == "" {
			fatalf("language %s has empty %s", code, key)
		}
		for _, r := range value {
			if r < 0x20 || r > 0x7e {
				fatalf("language %s field %s contains non-ASCII character %q; LCD builds must stay portable", code, key, r)
			}
		}
	}
}

func renderLanguageBlock(version, code string, pack stringsPack) string {
	q := func(value string) string { return fmt.Sprintf("%q", value) }
	return strings.Join([]string{
		beginMarker,
		"#define ZAPPER_FIRMWARE_VERSION " + q(version),
		"#define ZAPPER_LANG_CODE " + q(code),
		"#define UI_PROGRAM_FMT " + q(pack.ProgramFmt),
		"#define UI_MANUAL " + q(pack.Manual),
		"#define UI_STEP_TIME_FMT " + q(pack.StepTimeFmt),
		"#define UI_TIME_FMT " + q(pack.TimeFmt),
		"#define UI_START_MENU " + q(pack.StartMenu),
		"#define UI_USB_READY_FMT " + q(pack.USBReadyFmt),
		"#define UI_START_CANCEL " + q(pack.StartCancel),
		"#define UI_STOP " + q(pack.Stop),
		"#define UI_SESSION_DONE " + q(pack.SessionDone),
		"#define UI_CLICK_MENU " + q(pack.ClickMenu),
		endMarker,
	}, "\n")
}

func must(err error) {
	if err != nil { panic(err) }
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "firmware_i18n: "+format+"\n", args...)
	os.Exit(1)
}
