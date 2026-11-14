package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// Każdy klucz komunikatu urządzenia musi istnieć w katalogu TEXT frontendu z
// identycznym polskim szablonem — inaczej interfejs nie przetłumaczy ani stanu
// (message_key), ani błędu odrzuconego wywołania (dopasowanie po szablonie).
func TestDeviceMessageKeysMatchFrontendCatalog(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("web", "i18n.js"))
	if err != nil {
		t.Fatal(err)
	}
	for key, polish := range deviceMessages {
		pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `: \[("(?:[^"\\]|\\.)*"), "(?:[^"\\]|\\.)*"\]`)
		match := pattern.FindSubmatch(source)
		if match == nil {
			t.Errorf("klucz %s nie istnieje w TEXT (app/web/i18n.js)", key)
			continue
		}
		frontend, err := strconv.Unquote(string(match[1]))
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if frontend != polish {
			t.Errorf("%s: szablon frontendu %q różni się od backendu %q", key, frontend, polish)
		}
	}
}

func TestDeviceStatusCarriesMessageKeyAndParams(t *testing.T) {
	manager := &DeviceManager{
		connectionID: 4,
		status:       DeviceStatus{Connected: true, State: "connecting"},
	}
	manager.handleLine("READY ZAPPER_V5 5.1.0", 4)
	status := manager.Status()
	if status.MessageKey != "devBoardReady" || status.Message != "Płytka gotowa" {
		t.Fatalf("READY: key=%q message=%q", status.MessageKey, status.Message)
	}

	manager.handleLine("ERR BAD_STEP", 4)
	status = manager.Status()
	if status.MessageKey != "devBoardError" || status.LastError != "BAD_STEP" || status.LastErrorKey != "" {
		t.Fatalf("ERR: %#v", status)
	}

	// Linia protokołu bez własnego komunikatu zostaje surowa i bez klucza.
	manager.handleLine("STATE RUNNING 1 2 7830 1000", 4)
	status = manager.Status()
	if status.MessageKey != "" || status.Message != "STATE RUNNING 1 2 7830 1000" {
		t.Fatalf("surowa linia: key=%q message=%q", status.MessageKey, status.Message)
	}

	if err := manager.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(); status.MessageKey != "devDisconnected" {
		t.Fatalf("Disconnect: key=%q", status.MessageKey)
	}

	fresh := NewDeviceManager(t.TempDir(), nil)
	if status := fresh.Status(); status.MessageKey != "devSelectPort" || status.Message != "Wybierz port urządzenia" {
		t.Fatalf("stan początkowy: %#v", status)
	}
}

func TestDeviceErrorsCarryKeyParamsAndPolishText(t *testing.T) {
	err := validateDeviceSteps([]DeviceStep{{FrequencyMilliHz: 7_830, DurationSeconds: 1}, {FrequencyMilliHz: 1, DurationSeconds: 1}})
	var deviceErr *DeviceError
	if !errors.As(err, &deviceErr) {
		t.Fatalf("oczekiwano DeviceError, jest %T: %v", err, err)
	}
	if deviceErr.Key != "devErrStepFrequency" || deviceErr.Params["step"] != "2" {
		t.Fatalf("klucz/parametry: %#v", deviceErr)
	}
	if got := err.Error(); got != "krok 2 ma częstotliwość poza zakresem 0,1 Hz–4 MHz" {
		t.Fatalf("polski tekst: %q", got)
	}

	tooMany := validateDeviceSteps(make([]DeviceStep, deviceMaxSteps+1))
	if !errors.As(tooMany, &deviceErr) || deviceErr.Params["max"] != strconv.Itoa(deviceMaxSteps) {
		t.Fatalf("za dużo kroków: %#v", tooMany)
	}

	// Błąd zapisu na port: last_error ma klucz, parametry i oryginalny błąd sterownika.
	manager := &DeviceManager{
		port:   &fakeSerialConnection{err: errors.New("access denied")},
		status: DeviceStatus{Connected: true, Ready: true, State: "idle"},
	}
	_, startErr := manager.Start("session-1", "Test", []DeviceStep{{FrequencyMilliHz: 1_000, DurationSeconds: 1}})
	if !errors.As(startErr, &deviceErr) || deviceErr.Key != "devErrSendCommand" {
		t.Fatalf("błąd startu: %#v", startErr)
	}
	status := manager.Status()
	if status.MessageKey != "devSendFailed" || status.LastErrorKey != "devErrSendCommand" {
		t.Fatalf("status po błędzie: %#v", status)
	}
	if status.LastErrorParams["command"] != `"CLEAR"` || status.LastErrorParams["error"] != "access denied" {
		t.Fatalf("parametry błędu: %#v", status.LastErrorParams)
	}
	if status.LastError != `błąd wysyłania "CLEAR": access denied` {
		t.Fatalf("polski last_error: %q", status.LastError)
	}

	if _, err := (&DeviceManager{}).Connect("  "); !errors.As(err, &deviceErr) || deviceErr.Key != "devErrSelectCom" {
		t.Fatalf("pusty port: %#v", err)
	}
}
