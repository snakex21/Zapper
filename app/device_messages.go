package main

import (
	"errors"
	"strings"
)

// deviceMessages to katalog komunikatów stanu urządzenia wysyłanych do
// interfejsu. Backend przekazuje klucz i parametry (message_key,
// message_params), a frontend tłumaczy je przez ZapperI18n. Polski tekst
// zostaje w polu message dla logów i zgodności wstecznej. Te same klucze i
// polskie szablony muszą istnieć w TEXT w app/web/i18n.js — pilnuje tego
// TestDeviceMessageKeysMatchFrontendCatalog.
var deviceMessages = map[string]string{
	// Stan urządzenia.
	"devSelectPort":          "Wybierz port urządzenia",
	"devWaitingReady":        "Oczekiwanie na READY po resecie płytki",
	"devDisconnected":        "Urządzenie odłączone",
	"devSendingSession":      "Wysyłanie sesji do płytki",
	"devStoppingSession":     "Zatrzymywanie sesji",
	"devPausingSession":      "Wstrzymywanie sesji",
	"devSendFailed":          "Nie udało się wysłać sesji",
	"devCommLost":            "Przerwana komunikacja — ponawiam połączenie",
	"devPortRecovered":       "Port odzyskany — czekam na płytkę",
	"devRecoveryFailed":      "Nie udało się automatycznie odzyskać połączenia",
	"devBoardReady":          "Płytka gotowa",
	"devBoardResetRearm":     "Płytka wróciła po resecie — przygotowuję sesję ponownie",
	"devStoppedOnBoard":      "Sesja zatrzymana na płytce",
	"devSessionArmed":        "Sesja gotowa — potwierdź ją na płytce",
	"devTimeoutRearm":        "Płytka zakończyła oczekiwanie — przygotowuję sesję ponownie",
	"devCancelledTimeout":    "Sesja anulowana po czasie oczekiwania",
	"devCancelledOnBoard":    "Sesja anulowana na płytce",
	"devSessionDone":         "Sesja zakończona",
	"devBoardError":          "Płytka zgłosiła błąd",
	"devConnectionRecovered": "Połączenie odzyskane — sprawdzam oczekującą sesję",
	"devConnectionActive":    "Połączenie aktywne",
	"devConfirmedStarting":   "Potwierdzenie odebrane — uruchamianie",
	// Błędy operacji na urządzeniu.
	"devErrListPorts":          "nie można odczytać portów COM: {error}",
	"devErrSelectCom":          "wybierz port COM",
	"devErrOpenPort":           "nie można otworzyć {port}: {error}",
	"devErrNotConnected":       "płytka nie jest połączona",
	"devErrNotReady":           "płytka nie zgłosiła jeszcze READY",
	"devErrBusy":               "płytka ma już przygotowaną lub trwającą sesję",
	"devErrNothingToPause":     "nie trwa sesja profilu, którą można wstrzymać",
	"devErrNotRunning":         "sesja nie jest obecnie uruchomiona",
	"devErrRemainingUnknown":   "nie udało się ustalić pozostałej części sesji",
	"devErrNoSteps":            "sesja nie ma kroków dla urządzenia",
	"devErrTooManySteps":       "urządzenie obsługuje maksymalnie {max} kroków",
	"devErrStepFrequency":      "krok {step} ma częstotliwość poza zakresem 0,1 Hz–4 MHz",
	"devErrStepDuration":       "krok {step} ma czas poza zakresem 1 s–24 h",
	"devErrSendCommand":        "błąd wysyłania {command}: {error}",
	"devErrSessionAlreadyDone": "ta sesja jest już wykonana",
	"devErrSessionUnavailable": "sesja jest jeszcze niedostępna: {reason}",
	"devErrNoManualPause":      "brak wstrzymanej sesji ręcznej",
}

// deviceText renderuje polski szablon komunikatu z parametrami {nazwa}.
func deviceText(key string, params map[string]string) string {
	text, ok := deviceMessages[key]
	if !ok {
		return key
	}
	for name, value := range params {
		text = strings.ReplaceAll(text, "{"+name+"}", value)
	}
	return text
}

// DeviceError to błąd operacji urządzenia z kluczem tłumaczenia. Error()
// zwraca polski tekst (logi, errors.log); interfejs tłumaczy go na podstawie
// tego samego szablonu w katalogu TEXT.
type DeviceError struct {
	Key    string
	Params map[string]string
	Err    error
}

func (e *DeviceError) Error() string {
	return deviceText(e.Key, e.Params)
}

func (e *DeviceError) Unwrap() error {
	return e.Err
}

func newDeviceError(key string, params map[string]string) *DeviceError {
	return &DeviceError{Key: key, Params: params}
}

func wrapDeviceError(key string, params map[string]string, err error) *DeviceError {
	if params == nil {
		params = map[string]string{}
	}
	if err != nil {
		params["error"] = err.Error()
	}
	return &DeviceError{Key: key, Params: params, Err: err}
}

// setMessage ustawia komunikat stanu: klucz, parametry i polski tekst.
// Wywołujący musi trzymać d.mu.
func (d *DeviceManager) setMessage(key string, params map[string]string) {
	d.status.Message = deviceText(key, params)
	d.status.MessageKey = key
	d.status.MessageParams = params
}

// setRawMessage zapisuje surową linię protokołu z płytki (bez tłumaczenia).
// Wywołujący musi trzymać d.mu.
func (d *DeviceManager) setRawMessage(line string) {
	d.status.Message = line
	d.status.MessageKey = ""
	d.status.MessageParams = nil
}

// setLastError zapisuje ostatni błąd; błędy DeviceError zachowują klucz,
// pozostałe (np. komunikat sterownika COM albo kod ERR z firmware) są surowe.
// Wywołujący musi trzymać d.mu.
func (d *DeviceManager) setLastError(err error) {
	d.status.LastErrorKey = ""
	d.status.LastErrorParams = nil
	if err == nil {
		d.status.LastError = ""
		return
	}
	d.status.LastError = err.Error()
	var deviceErr *DeviceError
	if errors.As(err, &deviceErr) {
		d.status.LastErrorKey = deviceErr.Key
		d.status.LastErrorParams = deviceErr.Params
	}
}
