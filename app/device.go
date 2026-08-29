package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

const (
	deviceBaudRate       = 115200
	deviceMaxSteps       = 12
	deviceMinFrequency   = uint64(100)           // 0.1 Hz in milliHz
	deviceMaxFrequency   = uint64(4_000_000_000) // 4 MHz in milliHz
	deviceMaxDurationSec = uint32(24 * 60 * 60)
	deviceReconnectTries = 12
)

type serialConnection interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Drain() error
	ResetInputBuffer() error
	ResetOutputBuffer() error
	SetReadTimeout(time.Duration) error
	Close() error
}

type DevicePreferences struct {
	Port string `json:"port"`
}

type DeviceStatus struct {
	Connected          bool   `json:"connected"`
	Ready              bool   `json:"ready"`
	Port               string `json:"port"`
	PreferredPort      string `json:"preferred_port"`
	State              string `json:"state"`
	Firmware           string `json:"firmware"`
	Message            string `json:"message"`
	LastError          string `json:"last_error"`
	FrequencyMilliHz   uint64 `json:"frequency_millihz"`
	RemainingMS        uint32 `json:"remaining_ms"`
	StepIndex          int    `json:"step_index"`
	StepCount          int    `json:"step_count"`
	ActiveProfile      string `json:"active_profile"`
	ActiveSessionID    string `json:"active_session_id"`
	ManualPaused       bool   `json:"manual_paused"`
	ManualPauseSeconds uint32 `json:"manual_pause_seconds"`
	UpdatedAt          string `json:"updated_at"`
}

type DevicePauseState struct {
	SessionID        string       `json:"session_id"`
	ProfileName      string       `json:"profile_name"`
	RemainingSteps   []DeviceStep `json:"remaining_steps"`
	RemainingSeconds uint32       `json:"remaining_seconds"`
}

type DevicePauseResult struct {
	Status   DeviceStatus      `json:"status"`
	Snapshot *Snapshot         `json:"snapshot,omitempty"`
	Pause    *DevicePauseState `json:"pause,omitempty"`
}

type DeviceManager struct {
	mu                 sync.Mutex
	writeMu            sync.Mutex
	port               serialConnection
	status             DeviceStatus
	preferences        DevicePreferences
	preferencesPath    string
	connectionID       uint64
	onSessionDone      func(string)
	onSessionCancelled func(string)
	openPort           func(string, *serial.Mode) (serialConnection, error)
	activeSteps        []DeviceStep
	originalSteps      []DeviceStep
	manualPause        []DeviceStep
	reconnectDelay     time.Duration
}

func NewDeviceManager(appDirectory string, onSessionDone func(string)) *DeviceManager {
	dataDirectory := filepath.Join(appDirectory, "data")
	_ = prepareDataDirectory(appDirectory, dataDirectory)
	manager := &DeviceManager{
		preferencesPath: filepath.Join(dataDirectory, "device.json"),
		onSessionDone:   onSessionDone,
		reconnectDelay:  750 * time.Millisecond,
		openPort: func(name string, mode *serial.Mode) (serialConnection, error) {
			return serial.Open(name, mode)
		},
		status: DeviceStatus{State: "disconnected", Message: "Wybierz port urządzenia"},
	}
	if err := loadJSON(manager.preferencesPath, &manager.preferences); err == nil {
		manager.status.PreferredPort = manager.preferences.Port
	}
	return manager
}

func (d *DeviceManager) ListPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("nie można odczytać portów COM: %w", err)
	}
	ports = nonNilStrings(ports)
	sort.Strings(ports)
	return ports, nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (d *DeviceManager) Connect(portName string) (DeviceStatus, error) {
	portName = strings.TrimSpace(portName)
	if portName == "" {
		return d.Status(), errors.New("wybierz port COM")
	}
	_ = d.Disconnect()

	mode := &serial.Mode{BaudRate: deviceBaudRate, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit}
	port, err := d.openPort(portName, mode)
	if err != nil {
		return d.Status(), fmt.Errorf("nie można otworzyć %s: %w", portName, err)
	}
	if err := port.SetReadTimeout(300 * time.Millisecond); err != nil {
		_ = port.Close()
		return d.Status(), err
	}
	_ = port.ResetInputBuffer()
	_ = port.ResetOutputBuffer()

	d.mu.Lock()
	d.connectionID++
	connectionID := d.connectionID
	d.port = port
	d.preferences.Port = portName
	d.status = DeviceStatus{
		Connected:     true,
		Port:          portName,
		PreferredPort: portName,
		State:         "connecting",
		Message:       "Oczekiwanie na READY po resecie płytki",
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}
	d.mu.Unlock()
	if err := saveJSONAtomic(d.preferencesPath, d.preferences); err != nil {
		_ = d.Disconnect()
		return d.Status(), err
	}

	go d.readLoop(port, connectionID)
	go func() {
		time.Sleep(1800 * time.Millisecond)
		d.mu.Lock()
		stillCurrent := d.connectionID == connectionID && d.port == port
		d.mu.Unlock()
		if stillCurrent {
			_ = d.sendCommand("PING")
		}
	}()
	return d.Status(), nil
}

func (d *DeviceManager) Disconnect() error {
	d.mu.Lock()
	port := d.port
	d.connectionID++
	d.port = nil
	d.status.Connected = false
	d.status.Ready = false
	d.status.State = "disconnected"
	d.status.Message = "Urządzenie odłączone"
	d.status.ActiveProfile = ""
	d.status.ActiveSessionID = ""
	d.status.UpdatedAt = time.Now().Format(time.RFC3339)
	d.manualPause = nil
	d.mu.Unlock()
	if port != nil {
		return port.Close()
	}
	return nil
}

func (d *DeviceManager) Status() DeviceStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	status := d.status
	if status.PreferredPort == "" {
		status.PreferredPort = d.preferences.Port
	}
	if len(d.manualPause) > 0 && status.State == "idle" {
		status.ManualPaused = true
		for _, step := range d.manualPause {
			status.ManualPauseSeconds += step.DurationSeconds
		}
	}
	return status
}

// ManualResumeSteps zwraca kroki do wznowienia wstrzymanej sesji ręcznej
// (restart=false: reszta sesji; restart=true: pełna sesja od początku).
func (d *DeviceManager) ManualResumeSteps(restart bool) ([]DeviceStep, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if restart {
		if len(d.originalSteps) == 0 {
			return nil, false
		}
		return append([]DeviceStep(nil), d.originalSteps...), true
	}
	if len(d.manualPause) == 0 {
		return nil, false
	}
	return append([]DeviceStep(nil), d.manualPause...), true
}

func (d *DeviceManager) ClearManualPause() {
	d.mu.Lock()
	d.manualPause = nil
	d.mu.Unlock()
}

func (d *DeviceManager) Start(sessionID, profileName string, steps []DeviceStep) (DeviceStatus, error) {
	if err := validateDeviceSteps(steps); err != nil {
		return d.Status(), err
	}
	d.mu.Lock()
	if !d.status.Connected || d.port == nil {
		d.mu.Unlock()
		return d.Status(), errors.New("płytka nie jest połączona")
	}
	if !d.status.Ready {
		d.mu.Unlock()
		return d.Status(), errors.New("płytka nie zgłosiła jeszcze READY")
	}
	if d.status.State == "starting" || d.status.State == "armed" || d.status.State == "running" {
		d.mu.Unlock()
		return d.Status(), errors.New("płytka ma już przygotowaną lub trwającą sesję")
	}
	// Sesja ręczna nie ma identyfikatora planu — syntetyczne ID pozwala
	// wstrzymać ją i wznowić tak samo jak sesję profilu ("pauza jak w filmie").
	if strings.TrimSpace(sessionID) == "" {
		sessionID = fmt.Sprintf("manual_%d", time.Now().UnixNano())
	}
	d.status.ActiveProfile = strings.TrimSpace(profileName)
	d.status.ActiveSessionID = sessionID
	d.activeSteps = append([]DeviceStep(nil), steps...)
	d.originalSteps = append([]DeviceStep(nil), steps...)
	d.manualPause = nil
	d.status.State = "starting"
	d.status.StepCount = len(steps)
	d.status.Message = "Wysyłanie sesji do płytki"
	d.status.LastError = ""
	d.mu.Unlock()

	if err := d.sendCommand("CLEAR"); err != nil {
		return d.failStart(err)
	}
	for _, step := range steps {
		command := fmt.Sprintf("STEP %d %d", step.FrequencyMilliHz, uint64(step.DurationSeconds)*1000)
		if err := d.sendCommand(command); err != nil {
			return d.failStart(err)
		}
	}
	if err := d.sendCommand("START"); err != nil {
		return d.failStart(err)
	}
	return d.Status(), nil
}

func (d *DeviceManager) Stop() (DeviceStatus, error) {
	d.mu.Lock()
	d.status.State = "stopping"
	d.status.Message = "Zatrzymywanie sesji"
	d.status.ActiveProfile = ""
	d.status.ActiveSessionID = ""
	d.activeSteps = nil
	d.manualPause = nil
	d.mu.Unlock()
	if err := d.sendCommand("STOP"); err != nil {
		return d.failStart(err)
	}
	return d.Status(), nil
}

func (d *DeviceManager) Pause() (DevicePauseState, DeviceStatus, error) {
	d.mu.Lock()
	isManual := strings.HasPrefix(d.status.ActiveSessionID, "manual_")
	if d.status.ActiveSessionID == "" {
		d.mu.Unlock()
		return DevicePauseState{}, d.Status(), errors.New("nie trwa sesja profilu, którą można wstrzymać")
	}
	if d.status.State != "running" && d.status.State != "armed" && d.status.State != "starting" {
		d.mu.Unlock()
		return DevicePauseState{}, d.Status(), errors.New("sesja nie jest obecnie uruchomiona")
	}
	pause := DevicePauseState{
		SessionID:   d.status.ActiveSessionID,
		ProfileName: d.status.ActiveProfile,
	}
	stepIndex := d.status.StepIndex - 1
	if stepIndex < 0 {
		stepIndex = 0
	}
	if stepIndex >= len(d.activeSteps) {
		stepIndex = len(d.activeSteps) - 1
	}
	if len(d.activeSteps) > 0 && stepIndex >= 0 {
		remaining := d.activeSteps[stepIndex].DurationSeconds
		if d.status.State == "running" && d.status.RemainingMS > 0 {
			remaining = (d.status.RemainingMS + 999) / 1000
		}
		if remaining > 0 {
			current := d.activeSteps[stepIndex]
			current.DurationSeconds = remaining
			pause.RemainingSteps = append(pause.RemainingSteps, current)
		}
		pause.RemainingSteps = append(pause.RemainingSteps, d.activeSteps[stepIndex+1:]...)
	}
	for _, step := range pause.RemainingSteps {
		pause.RemainingSeconds += step.DurationSeconds
	}
	d.status.State = "stopping"
	d.status.Message = "Wstrzymywanie sesji"
	d.status.ActiveProfile = ""
	d.status.ActiveSessionID = ""
	d.activeSteps = nil
	if isManual {
		d.manualPause = append([]DeviceStep(nil), pause.RemainingSteps...)
	}
	d.mu.Unlock()
	if len(pause.RemainingSteps) == 0 {
		return DevicePauseState{}, d.Status(), errors.New("nie udało się ustalić pozostałej części sesji")
	}
	if err := d.sendCommand("STOP"); err != nil {
		status, failErr := d.failStart(err)
		return DevicePauseState{}, status, failErr
	}
	return pause, d.Status(), nil
}

func (d *DeviceManager) failStart(err error) (DeviceStatus, error) {
	d.mu.Lock()
	d.status.State = "error"
	d.status.LastError = err.Error()
	d.status.Message = "Nie udało się wysłać sesji"
	d.status.ActiveProfile = ""
	d.status.ActiveSessionID = ""
	d.activeSteps = nil
	d.mu.Unlock()
	return d.Status(), err
}

func validateDeviceSteps(steps []DeviceStep) error {
	if len(steps) == 0 {
		return errors.New("sesja nie ma kroków dla urządzenia")
	}
	if len(steps) > deviceMaxSteps {
		return fmt.Errorf("urządzenie obsługuje maksymalnie %d kroków", deviceMaxSteps)
	}
	for index, step := range steps {
		if step.FrequencyMilliHz < deviceMinFrequency || step.FrequencyMilliHz > deviceMaxFrequency {
			return fmt.Errorf("krok %d ma częstotliwość poza zakresem 0,1 Hz–4 MHz", index+1)
		}
		if step.DurationSeconds == 0 || step.DurationSeconds > deviceMaxDurationSec {
			return fmt.Errorf("krok %d ma czas poza zakresem 1 s–24 h", index+1)
		}
	}
	return nil
}

func (d *DeviceManager) sendCommand(command string) error {
	d.mu.Lock()
	port := d.port
	connected := d.status.Connected
	d.mu.Unlock()
	if !connected || port == nil {
		return errors.New("płytka nie jest połączona")
	}
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	if _, err := port.Write([]byte(command + "\n")); err != nil {
		return fmt.Errorf("błąd wysyłania %q: %w", command, err)
	}
	return port.Drain()
}

func (d *DeviceManager) readLoop(port serialConnection, connectionID uint64) {
	buffer := make([]byte, 128)
	pending := ""
	for {
		count, err := port.Read(buffer)
		if count > 0 {
			pending += string(buffer[:count])
			for {
				lineEnd := strings.IndexByte(pending, '\n')
				if lineEnd < 0 {
					if len(pending) > 512 {
						pending = ""
					}
					break
				}
				line := strings.TrimSpace(pending[:lineEnd])
				pending = pending[lineEnd+1:]
				if line != "" {
					d.handleLine(line, connectionID)
				}
			}
		}
		if err != nil {
			d.handleReadFailure(port, connectionID, err)
			return
		}
	}
}

// handleReadFailure nie uznaje krótkiego błędu sterownika USB za ręczne
// rozłączenie. Nano/CH340 potrafi na moment zgubić uchwyt portu przy resecie;
// aplikacja zachowuje wybrany COM i próbuje otworzyć go ponownie.
func (d *DeviceManager) handleReadFailure(port serialConnection, connectionID uint64, readErr error) {
	d.mu.Lock()
	if d.connectionID != connectionID || d.port != port {
		d.mu.Unlock()
		return
	}
	portName := d.status.Port
	d.port = nil
	d.status.Connected = false
	d.status.Ready = false
	d.status.State = "reconnecting"
	d.status.LastError = readErr.Error()
	d.status.Message = "Przerwana komunikacja — ponawiam połączenie"
	d.status.UpdatedAt = time.Now().Format(time.RFC3339)
	d.mu.Unlock()
	_ = port.Close()
	go d.reconnectLoop(portName, connectionID)
}

func (d *DeviceManager) reconnectLoop(portName string, connectionID uint64) {
	delay := d.reconnectDelay
	if delay <= 0 {
		delay = 750 * time.Millisecond
	}
	for attempt := 1; attempt <= deviceReconnectTries; attempt++ {
		time.Sleep(delay)
		d.mu.Lock()
		stillWanted := d.connectionID == connectionID && d.port == nil && d.status.State == "reconnecting"
		d.mu.Unlock()
		if !stillWanted {
			return
		}

		mode := &serial.Mode{BaudRate: deviceBaudRate, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit}
		port, err := d.openPort(portName, mode)
		if err != nil {
			continue
		}
		if err := port.SetReadTimeout(300 * time.Millisecond); err != nil {
			_ = port.Close()
			continue
		}
		_ = port.ResetInputBuffer()
		_ = port.ResetOutputBuffer()

		d.mu.Lock()
		if d.connectionID != connectionID || d.port != nil || d.status.State != "reconnecting" {
			d.mu.Unlock()
			_ = port.Close()
			return
		}
		d.port = port
		d.status.Connected = true
		d.status.Port = portName
		d.status.State = "connecting"
		d.status.LastError = ""
		d.status.Message = "Port odzyskany — czekam na płytkę"
		d.status.UpdatedAt = time.Now().Format(time.RFC3339)
		d.mu.Unlock()

		go d.readLoop(port, connectionID)
		go func(recoveredPort serialConnection) {
			time.Sleep(1800 * time.Millisecond)
			d.mu.Lock()
			current := d.connectionID == connectionID && d.port == recoveredPort
			d.mu.Unlock()
			if current {
				_ = d.sendCommand("PING")
				_ = d.sendCommand("STATUS")
			}
		}(port)
		return
	}

	d.mu.Lock()
	if d.connectionID == connectionID && d.port == nil && d.status.State == "reconnecting" {
		d.status.State = "error"
		d.status.Message = "Nie udało się automatycznie odzyskać połączenia"
		d.status.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	d.mu.Unlock()
}

func (d *DeviceManager) handleLine(line string, connectionID uint64) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	var completedSessionID string
	var cancelledSessionID string
	var rearmSessionID string
	var rearmProfile string
	var rearmSteps []DeviceStep
	d.mu.Lock()
	if d.connectionID != connectionID {
		d.mu.Unlock()
		return
	}
	d.status.UpdatedAt = time.Now().Format(time.RFC3339)
	d.status.Message = line
	switch fields[0] {
	case "READY":
		previousState := d.status.State
		previousSessionID := d.status.ActiveSessionID
		d.status.Ready = true
		d.status.Connected = true
		d.status.State = "idle"
		d.status.LastError = ""
		d.status.Message = "Płytka gotowa"
		if previousSessionID != "" && previousState == "connecting" && len(d.activeSteps) > 0 {
			// READY po automatycznym ponownym otwarciu COM oznacza reset Nano.
			// Sesja oczekująca nie może przez to zniknąć — wysyłamy ją ponownie.
			rearmSessionID = previousSessionID
			rearmProfile = d.status.ActiveProfile
			rearmSteps = append([]DeviceStep(nil), d.activeSteps...)
			d.status.ActiveProfile = ""
			d.status.ActiveSessionID = ""
			d.activeSteps = nil
			d.status.Message = "Płytka wróciła po resecie — przygotowuję sesję ponownie"
		} else if previousSessionID != "" && (previousState == "armed" || previousState == "starting" || previousState == "running" || previousState == "connecting") {
			cancelledSessionID = previousSessionID
			d.status.ActiveProfile = ""
			d.status.ActiveSessionID = ""
			d.activeSteps = nil
		}
		if len(fields) >= 3 {
			d.status.Firmware = strings.Join(fields[1:], " ")
		}
	case "STATE", "STATUS":
		previousState := d.status.State
		previousSessionID := d.status.ActiveSessionID
		if len(fields) >= 2 {
			d.status.State = strings.ToLower(fields[1])
		}
		if len(fields) >= 6 {
			d.status.StepIndex, _ = strconv.Atoi(fields[2])
			d.status.StepCount, _ = strconv.Atoi(fields[3])
			d.status.FrequencyMilliHz, _ = strconv.ParseUint(fields[4], 10, 64)
			remaining, _ := strconv.ParseUint(fields[5], 10, 32)
			d.status.RemainingMS = uint32(remaining)
		}
		if previousState == "running" && d.status.State == "idle" && previousSessionID != "" {
			cancelledSessionID = previousSessionID
			d.status.ActiveProfile = ""
			d.status.ActiveSessionID = ""
			d.activeSteps = nil
			d.status.Message = "Sesja zatrzymana na płytce"
		}
	case "ARMED":
		d.status.Ready = true
		d.status.State = "armed"
		d.status.StepIndex = 0
		d.status.FrequencyMilliHz = 0
		d.status.RemainingMS = 0
		d.status.Message = "Sesja gotowa — potwierdź ją na płytce"
		if len(fields) >= 2 {
			d.status.StepCount, _ = strconv.Atoi(fields[1])
		}
	case "CANCELLED":
		isTimeout := len(fields) >= 2 && fields[1] == "TIMEOUT"
		if isTimeout && d.status.ActiveSessionID != "" && len(d.activeSteps) > 0 {
			rearmSessionID = d.status.ActiveSessionID
			rearmProfile = d.status.ActiveProfile
			rearmSteps = append([]DeviceStep(nil), d.activeSteps...)
		}
		d.status.State = "idle"
		d.status.RemainingMS = 0
		d.status.FrequencyMilliHz = 0
		d.status.ActiveProfile = ""
		d.status.ActiveSessionID = ""
		d.activeSteps = nil
		d.status.LastError = ""
		if isTimeout && len(rearmSteps) > 0 {
			d.status.Message = "Płytka zakończyła oczekiwanie — przygotowuję sesję ponownie"
		} else if isTimeout {
			d.status.Message = "Sesja anulowana po czasie oczekiwania"
		} else {
			d.status.Message = "Sesja anulowana na płytce"
		}
	case "DONE":
		d.status.State = "done"
		d.status.RemainingMS = 0
		d.status.Message = "Sesja zakończona"
		completedSessionID = d.status.ActiveSessionID
		d.status.ActiveProfile = ""
		d.status.ActiveSessionID = ""
		d.activeSteps = nil
	case "ERR":
		d.status.State = "error"
		d.status.LastError = strings.TrimSpace(strings.TrimPrefix(line, "ERR"))
		d.status.Message = "Płytka zgłosiła błąd"
	case "OK":
		if len(fields) >= 2 && fields[1] == "PONG" {
			d.status.Ready = true
			if d.status.State == "connecting" {
				if d.status.ActiveSessionID != "" {
					d.status.State = "starting"
				} else {
					d.status.State = "idle"
				}
			}
			if len(fields) >= 3 {
				d.status.Firmware = strings.Join(fields[2:], " ")
			}
			if d.status.ActiveSessionID != "" {
				d.status.Message = "Połączenie odzyskane — sprawdzam oczekującą sesję"
			} else {
				d.status.Message = "Połączenie aktywne"
			}
		} else if len(fields) >= 2 && fields[1] == "CONFIRMED" {
			d.status.Message = "Potwierdzenie odebrane — uruchamianie"
		}
	}
	d.mu.Unlock()

	if completedSessionID != "" && d.onSessionDone != nil {
		d.onSessionDone(completedSessionID)
	}
	if cancelledSessionID != "" && d.onSessionCancelled != nil {
		d.onSessionCancelled(cancelledSessionID)
	}
	if len(rearmSteps) > 0 {
		go func() {
			// Starsze firmware wysyła po CANCELLED jeszcze STATE IDLE, a po
			// resecie portu READY. Krótka zwłoka pozwala odebrać komunikat
			// kończący poprzedni stan przed ponownym uzbrojeniem.
			time.Sleep(150 * time.Millisecond)
			_, _ = d.Start(rearmSessionID, rearmProfile, rearmSteps)
		}()
	}
}
