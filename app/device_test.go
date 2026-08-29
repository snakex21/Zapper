package main

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
)

type fakeSerialConnection struct {
	writes []string
	err    error
}

type blockingSerialConnection struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingSerialConnection() *blockingSerialConnection {
	return &blockingSerialConnection{closed: make(chan struct{})}
}

func (f *blockingSerialConnection) Read([]byte) (int, error) {
	<-f.closed
	return 0, errors.New("port closed")
}
func (f *blockingSerialConnection) Write(data []byte) (int, error) { return len(data), nil }
func (f *blockingSerialConnection) Drain() error                   { return nil }
func (f *blockingSerialConnection) ResetInputBuffer() error        { return nil }
func (f *blockingSerialConnection) ResetOutputBuffer() error       { return nil }
func (f *blockingSerialConnection) SetReadTimeout(time.Duration) error {
	return nil
}
func (f *blockingSerialConnection) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func (f *fakeSerialConnection) Read([]byte) (int, error)           { return 0, f.err }
func (f *fakeSerialConnection) Drain() error                       { return f.err }
func (f *fakeSerialConnection) ResetInputBuffer() error            { return nil }
func (f *fakeSerialConnection) ResetOutputBuffer() error           { return nil }
func (f *fakeSerialConnection) SetReadTimeout(time.Duration) error { return nil }
func (f *fakeSerialConnection) Close() error                       { return nil }
func (f *fakeSerialConnection) Write(data []byte) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.writes = append(f.writes, string(data))
	return len(data), nil
}

func TestValidateDeviceSteps(t *testing.T) {
	valid := []DeviceStep{
		{FrequencyMilliHz: 7_830, DurationSeconds: 420},
		{FrequencyMilliHz: 30_000_000, DurationSeconds: 600},
	}
	if err := validateDeviceSteps(valid); err != nil {
		t.Fatalf("poprawne kroki odrzucone: %v", err)
	}

	tests := []struct {
		name  string
		steps []DeviceStep
	}{
		{"brak kroków", nil},
		{"za niska częstotliwość", []DeviceStep{{FrequencyMilliHz: 99, DurationSeconds: 1}}},
		{"za wysoka częstotliwość", []DeviceStep{{FrequencyMilliHz: 4_000_000_001, DurationSeconds: 1}}},
		{"zerowy czas", []DeviceStep{{FrequencyMilliHz: 100, DurationSeconds: 0}}},
		{"za dużo kroków", make([]DeviceStep, deviceMaxSteps+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateDeviceSteps(test.steps); err == nil {
				t.Fatal("oczekiwano błędu walidacji")
			}
		})
	}
}

func TestListPortsReturnsEmptyArrayInsteadOfNil(t *testing.T) {
	ports := nonNilStrings(nil)
	if ports == nil || len(ports) != 0 {
		t.Fatalf("oczekiwano pustej, nie-nilowej listy: %#v", ports)
	}
}

func TestDeviceStartWritesMilliHzProtocol(t *testing.T) {
	port := &fakeSerialConnection{}
	manager := &DeviceManager{
		port:   port,
		status: DeviceStatus{Connected: true, Ready: true, State: "idle"},
	}
	steps := []DeviceStep{
		{FrequencyMilliHz: 7_830, DurationSeconds: 420},
		{FrequencyMilliHz: 30_000_000, DurationSeconds: 60},
	}
	if _, err := manager.Start("session-1", "Test", steps); err != nil {
		t.Fatal(err)
	}
	want := "CLEAR\nSTEP 7830 420000\nSTEP 30000000 60000\nSTART\n"
	if got := strings.Join(port.writes, ""); got != want {
		t.Fatalf("nieprawidłowe komendy:\n%s\nchciano:\n%s", got, want)
	}
}

func TestDeviceParsesBoardStateAndCompletion(t *testing.T) {
	completed := ""
	manager := &DeviceManager{
		connectionID:  9,
		onSessionDone: func(sessionID string) { completed = sessionID },
		status:        DeviceStatus{Connected: true, ActiveProfile: "Profil A", ActiveSessionID: "session-a"},
	}
	manager.handleLine("READY ZAPPER_V5 5.0.1", 9)
	manager.handleLine("ARMED 3", 9)
	if status := manager.Status(); status.State != "armed" || status.StepCount != 3 || !status.Ready {
		t.Fatalf("nieprawidłowy stan oczekiwania na potwierdzenie: %#v", status)
	}
	manager.handleLine("STATE RUNNING 2 3 30000000 419000", 9)
	status := manager.Status()
	if !status.Ready || status.State != "running" || status.StepIndex != 2 || status.StepCount != 3 {
		t.Fatalf("nieprawidłowo odczytany stan: %#v", status)
	}
	if status.FrequencyMilliHz != 30_000_000 || status.RemainingMS != 419_000 {
		t.Fatalf("nieprawidłowe dane kroku: %#v", status)
	}
	manager.handleLine("DONE", 9)
	if completed != "session-a" {
		t.Fatalf("nie oznaczono sesji po DONE: %q", completed)
	}
	if manager.Status().State != "done" {
		t.Fatal("stan po DONE powinien mieć wartość done")
	}
}

func TestDeviceStartReportsWriteFailure(t *testing.T) {
	manager := &DeviceManager{
		port:   &fakeSerialConnection{err: errors.New("test write failure")},
		status: DeviceStatus{Connected: true, Ready: true, State: "idle"},
	}
	if _, err := manager.Start("session-1", "Test", []DeviceStep{{FrequencyMilliHz: 1_000, DurationSeconds: 1}}); err == nil {
		t.Fatal("oczekiwano błędu zapisu")
	}
	if manager.Status().State != "error" {
		t.Fatal("błąd wysyłania powinien ustawić stan error")
	}
}

func TestDevicePauseKeepsOnlyRemainingSteps(t *testing.T) {
	port := &fakeSerialConnection{}
	manager := &DeviceManager{
		port:   port,
		status: DeviceStatus{Connected: true, Ready: true, State: "idle"},
	}
	steps := []DeviceStep{
		{FrequencyMilliHz: 30_000_000, DurationSeconds: 420},
		{FrequencyMilliHz: 7_830, DurationSeconds: 60},
	}
	if _, err := manager.Start("session-1", "Test", steps); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.status.State = "running"
	manager.status.StepIndex = 1
	manager.status.RemainingMS = 199_100
	manager.mu.Unlock()
	pause, _, err := manager.Pause()
	if err != nil {
		t.Fatal(err)
	}
	if pause.SessionID != "session-1" || pause.RemainingSeconds != 260 || len(pause.RemainingSteps) != 2 {
		t.Fatalf("nieprawidłowy zapis pauzy: %#v", pause)
	}
	if pause.RemainingSteps[0].DurationSeconds != 200 {
		t.Fatalf("bieżący krok nie został skrócony: %#v", pause.RemainingSteps[0])
	}
	if got := port.writes[len(port.writes)-1]; got != "STOP\n" {
		t.Fatalf("pauza nie wysłała STOP: %q", got)
	}
}

func TestOldFirmwareTimeoutRearmsSessionInsteadOfDroppingIt(t *testing.T) {
	port := &fakeSerialConnection{}
	steps := []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 420}}
	manager := &DeviceManager{
		port:         port,
		connectionID: 9,
		status: DeviceStatus{
			Connected:       true,
			Ready:           true,
			State:           "armed",
			ActiveProfile:   "Test",
			ActiveSessionID: "session-waiting",
		},
		activeSteps: steps,
	}
	manager.handleLine("CANCELLED TIMEOUT", 9)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if manager.Status().State == "starting" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	status := manager.Status()
	if status.State != "starting" || status.ActiveSessionID != "session-waiting" || !status.Connected {
		t.Fatalf("sesja nie została ponownie przygotowana po timeout: %#v", status)
	}
	if got := strings.Join(port.writes, ""); !strings.Contains(got, "CLEAR\nSTEP 30000000 420000\nSTART\n") {
		t.Fatalf("nie wysłano sesji ponownie: %q", got)
	}
}

func TestRecoveredPortKeepsPendingSessionUntilBoardStateArrives(t *testing.T) {
	manager := &DeviceManager{
		connectionID: 9,
		status: DeviceStatus{
			Connected:       true,
			State:           "connecting",
			ActiveProfile:   "Test",
			ActiveSessionID: "session-waiting",
		},
	}
	manager.handleLine("OK PONG ZAPPER_V5 5.0.2", 9)
	status := manager.Status()
	if status.State != "starting" || status.ActiveSessionID != "session-waiting" || !status.Ready {
		t.Fatalf("PONG po odzyskaniu COM zgubił oczekującą sesję: %#v", status)
	}
	manager.handleLine("STATUS ARMED 0 1 0 0", 9)
	if status = manager.Status(); status.State != "armed" || status.ActiveSessionID != "session-waiting" {
		t.Fatalf("stan płytki nie przywrócił ekranu oczekiwania: %#v", status)
	}
}

func TestBoardResetWhileWaitingRearmsPendingSession(t *testing.T) {
	port := &fakeSerialConnection{}
	steps := []DeviceStep{{FrequencyMilliHz: 30_000_000, DurationSeconds: 420}}
	manager := &DeviceManager{
		port:         port,
		connectionID: 9,
		status: DeviceStatus{
			Connected:       true,
			State:           "connecting",
			ActiveProfile:   "Test",
			ActiveSessionID: "session-reset",
		},
		activeSteps: steps,
	}
	manager.handleLine("READY ZAPPER_V5 5.0.2", 9)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if manager.Status().State == "starting" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status := manager.Status(); status.State != "starting" || status.ActiveSessionID != "session-reset" {
		t.Fatalf("reset płytki zgubił oczekującą sesję: %#v", status)
	}
}

func TestReadFailureAutomaticallyReopensSamePort(t *testing.T) {
	broken := &fakeSerialConnection{}
	recovered := newBlockingSerialConnection()
	manager := &DeviceManager{
		port:           broken,
		connectionID:   9,
		reconnectDelay: time.Millisecond,
		status: DeviceStatus{
			Connected:       true,
			Ready:           true,
			Port:            "COM3",
			State:           "armed",
			ActiveSessionID: "session-waiting",
		},
		openPort: func(name string, _ *serial.Mode) (serialConnection, error) {
			if name != "COM3" {
				t.Fatalf("próba odzyskania innego portu: %s", name)
			}
			return recovered, nil
		},
	}
	manager.handleReadFailure(broken, 9, errors.New("chwilowy błąd USB"))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status := manager.Status()
		if status.Connected && status.State == "connecting" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	status := manager.Status()
	if !status.Connected || status.State != "connecting" || status.Port != "COM3" || status.ActiveSessionID != "session-waiting" {
		t.Fatalf("port nie został automatycznie odzyskany: %#v", status)
	}
	if err := manager.Disconnect(); err != nil {
		t.Fatal(err)
	}
}
