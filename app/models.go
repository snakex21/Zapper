package main

import (
	"bytes"
	"encoding/json"
)

type Config struct {
	StartDate string    `json:"start_date"`
	Profiles  []Profile `json:"profiles"`
}

type Profile struct {
	ID        string  `json:"id,omitempty"`
	PersonID  string  `json:"person_id,omitempty"`
	RunID     string  `json:"run_id,omitempty"`
	StartDate string  `json:"start_date,omitempty"`
	Name      string  `json:"name"`
	Phases    []Phase `json:"phases"`
}

type Person struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type PersonStore struct {
	SchemaVersion int      `json:"schema_version"`
	Persons       []Person `json:"persons"`
}

type Phase struct {
	ID           string               `json:"id,omitempty"`
	Name         string               `json:"name"`
	DurationDays int                  `json:"duration_days"`
	Mode         string               `json:"mode"`
	IntervalGap  int                  `json:"interval_gap,omitempty"`
	Program      string               `json:"program,omitempty"`
	Time         string               `json:"time,omitempty"`
	Note         string               `json:"note,omitempty"`
	DeviceSteps  []DeviceStep         `json:"device_steps,omitempty"`
	Scheduling   SessionScheduling    `json:"scheduling,omitempty"`
	Schedule     map[string]DailyPlan `json:"schedule,omitempty"`
}

type DailyPlan struct {
	Frequency   string            `json:"freq"`
	Time        string            `json:"time"`
	Note        string            `json:"note"`
	DeviceSteps []DeviceStep      `json:"device_steps,omitempty"`
	Scheduling  SessionScheduling `json:"scheduling,omitempty"`
}

type SessionScheduling struct {
	ProgramID            string   `json:"program_id,omitempty"`
	Repetitions          int      `json:"repetitions,omitempty"`
	BreakBetweenMinutes  int      `json:"break_between_minutes,omitempty"`
	MinGapMinutes        int      `json:"min_gap_minutes,omitempty"`
	CooldownAfterMinutes int      `json:"cooldown_after_minutes,omitempty"`
	SameDayWith          []string `json:"same_day_with,omitempty"`
}

type DeviceStep struct {
	FrequencyMilliHz uint64 `json:"frequency_millihz"`
	DurationSeconds  uint32 `json:"duration_seconds"`
}

type Progress struct {
	StartDate      string                             `json:"start_date"`
	History        map[string]map[string]HistoryEntry `json:"history"`
	TrackingSince  string                             `json:"tracking_since,omitempty"`
	Completions    map[string]SessionCompletion       `json:"completions,omitempty"`
	PausedSessions map[string]PausedSession           `json:"paused_sessions,omitempty"`
	// DismissedSessions to zaległe sesje ODPUSZCZONE przez użytkownika.
	// To NIE jest wykonanie — nie liczy się do historii, statystyk ani kontekstu AI.
	// Trzymamy je w osobnej mapie właśnie po to, żeby nie dało się ich pomylić z SessionCompletion.
	DismissedSessions map[string]DismissedSession `json:"dismissed_sessions,omitempty"`
	// PartialRuns to ślady CZĘŚCIOWEGO wykonania sesji przerwanej przyciskiem
	// Zatrzymaj. To nie jest wykonanie (sesja nadal czeka na dokończenie), ale
	// użytkownik wykonał kawałek — wpis dokumentuje ile, ile zostało i kiedy.
	PartialRuns []PartialRun `json:"partial_runs,omitempty"`
}

// PartialRun dokumentuje częściowe wykonanie sesji przerwanej Zatrzymaj.
// SessionID wskazuje wciąż oczekującą sesję; RemainingSeconds to reszta,
// którą wznowienie wyśle na płytkę.
type PartialRun struct {
	SessionID        string `json:"session_id"`
	ProfileID        string `json:"profile_id"`
	PersonID         string `json:"person_id"`
	RunID            string `json:"run_id"`
	ProfileName      string `json:"profile_name"`
	ProgramID        string `json:"program_id,omitempty"`
	Therapy          string `json:"therapy"`
	Phase            string `json:"phase"`
	PlannedDate      string `json:"planned_date"`
	DoneSeconds      uint32 `json:"done_seconds"`
	TotalSeconds     uint32 `json:"total_seconds"`
	RemainingSeconds uint32 `json:"remaining_seconds"`
	RecordedAt       string `json:"recorded_at"`
}

type PausedSession struct {
	SessionID        string       `json:"session_id"`
	ProfileID        string       `json:"profile_id"`
	PersonID         string       `json:"person_id"`
	RunID            string       `json:"run_id"`
	ProfileName      string       `json:"profile_name"`
	PausedAt         string       `json:"paused_at"`
	FirstPausedAt    string       `json:"first_paused_at,omitempty"`
	RemainingSteps   []DeviceStep `json:"remaining_steps"`
	RemainingSeconds uint32       `json:"remaining_seconds"`
	// Recorded oznacza pauzę po przycisku Zatrzymaj: postęp zapisany w PartialRuns,
	// a ekran sesji (panel) ma pozostać zamknięty mimo oczekującego wznowienia.
	Recorded bool `json:"recorded,omitempty"`
}

// DismissedSession dokumentuje świadomą rezygnację z zaległej sesji.
// Zabieg się NIE odbył — pole istnieje wyłącznie po to, żeby zaległość
// przestała blokować profil, a historia pozostała prawdziwa.
type DismissedSession struct {
	SessionID      string `json:"session_id"`
	SessionGroupID string `json:"session_group_id,omitempty"`
	ProfileID      string `json:"profile_id"`
	PersonID       string `json:"person_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	ProfileName    string `json:"profile_name"`
	ProgramID      string `json:"program_id,omitempty"`
	Therapy        string `json:"therapy,omitempty"`
	Phase          string `json:"phase,omitempty"`
	PlannedDate    string `json:"planned_date"`
	DismissedAt    string `json:"dismissed_at"`
}

// ManualRun to ślad po ręcznym uruchomieniu płytki (widok Urządzenie → tryb ręczny).
// Celowo NIE jest to SessionCompletion: ręczny przebieg nie należy do żadnego planu,
// nie zamyka zaplanowanej sesji i nie może być liczony jako wykonanie planu.
type ManualRun struct {
	RunRecordID      string `json:"run_record_id"`
	StartedAt        string `json:"started_at"`
	FrequencyMilliHz uint64 `json:"frequency_millihz"`
	DurationSeconds  uint32 `json:"duration_seconds"`
	Source           string `json:"source"`
}

type ManualRunStore struct {
	Runs []ManualRun `json:"runs"`
}

type HistoryEntry struct {
	Done        bool   `json:"done"`
	Therapy     string `json:"therapy,omitempty"`
	Time        string `json:"time,omitempty"`
	Repetitions string `json:"repetitions,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Notes       string `json:"notes,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type SessionCompletion struct {
	SessionID      string            `json:"session_id"`
	SessionGroupID string            `json:"session_group_id,omitempty"`
	ProfileID      string            `json:"profile_id"`
	PersonID       string            `json:"person_id,omitempty"`
	RunID          string            `json:"run_id,omitempty"`
	ProfileName    string            `json:"profile_name"`
	ProgramID      string            `json:"program_id"`
	PlannedDate    string            `json:"planned_date"`
	CompletedAt    string            `json:"completed_at"`
	Therapy        string            `json:"therapy"`
	Time           string            `json:"time"`
	Repetitions    string            `json:"repetitions"`
	Phase          string            `json:"phase"`
	Notes          string            `json:"notes,omitempty"`
	Scheduling     SessionScheduling `json:"scheduling,omitempty"`
	// Split oznacza, że przebieg był dzielony pauzą i dokończony później.
	// FirstStartedAt to moment PIERWSZEJ pauzy tego przebiegu — bez tego
	// sesja wznowiona po tygodniu wyglądałaby w historii jak jedno ciągłe wykonanie.
	Split          bool   `json:"split,omitempty"`
	FirstStartedAt string `json:"first_started_at,omitempty"`
}

func (h *HistoryEntry) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("true")) || bytes.Equal(trimmed, []byte("false")) {
		return json.Unmarshal(trimmed, &h.Done)
	}
	type alias HistoryEntry
	return json.Unmarshal(trimmed, (*alias)(h))
}

type FrequencyEntry struct {
	Name        string `json:"name"`
	Frequency   string `json:"frequency"`
	Description string `json:"description"`
}

type DayPlan struct {
	SessionID         string            `json:"session_id,omitempty"`
	SessionGroupID    string            `json:"session_group_id,omitempty"`
	ProfileID         string            `json:"profile_id,omitempty"`
	PersonID          string            `json:"person_id,omitempty"`
	RunID             string            `json:"run_id,omitempty"`
	PhaseID           string            `json:"phase_id,omitempty"`
	Date              string            `json:"date"`
	PlannedDate       string            `json:"planned_date,omitempty"`
	DayName           string            `json:"day_name"`
	DayNumber         int               `json:"day_number"`
	ProfileName       string            `json:"profile_name"`
	PhaseName         string            `json:"phase_name"`
	Status            string            `json:"status"`
	Program           string            `json:"program"`
	Time              string            `json:"time"`
	Repetitions       string            `json:"repetitions"`
	Note              string            `json:"note"`
	Done              bool              `json:"done"`
	DeviceSteps       []DeviceStep      `json:"device_steps,omitempty"`
	Scheduling        SessionScheduling `json:"scheduling,omitempty"`
	RepeatIndex       int               `json:"repeat_index,omitempty"`
	RepeatCount       int               `json:"repeat_count,omitempty"`
	Overdue           bool              `json:"overdue,omitempty"`
	Available         bool              `json:"available"`
	AvailableAt       string            `json:"available_at,omitempty"`
	BlockedReason     string            `json:"blocked_reason,omitempty"`
	ExtensionDays     int               `json:"extension_days,omitempty"`
	Paused            bool              `json:"paused,omitempty"`
	RemainingSeconds  uint32            `json:"remaining_seconds,omitempty"`
	PausedRecorded    bool              `json:"paused_recorded,omitempty"`
	OlderPendingCount int               `json:"older_pending_count,omitempty"`
}

type ProfileSchedule struct {
	ProfileName string    `json:"profile_name"`
	Days        []DayPlan `json:"days"`
}

type AppMeta struct {
	DataDirectory string `json:"data_directory"`
	ProfilesFile  string `json:"profiles_file"`
	ProgressFile  string `json:"progress_file"`
	Version       string `json:"version"`
}

type ProfileRuntimeStatus struct {
	ProfileID     string `json:"profile_id"`
	ProfileName   string `json:"profile_name"`
	OverdueCount  int    `json:"overdue_count"`
	ExtensionDays int    `json:"extension_days"`
}

type ArchivedProfile struct {
	ArchiveID      string    `json:"archive_id"`
	FinishedAt     string    `json:"finished_at"`
	Reason         string    `json:"reason"`
	Profile        Profile   `json:"profile"`
	OverdueCount   int       `json:"overdue_count"`
	ExtensionDays  int       `json:"extension_days"`
	CompletedCount int       `json:"completed_count"`
	Progress       *Progress `json:"progress,omitempty"`
}

type ArchiveStore struct {
	Profiles []ArchivedProfile `json:"profiles"`
}

type Snapshot struct {
	Config                Config                 `json:"config"`
	Progress              Progress               `json:"progress"`
	Today                 []DayPlan              `json:"today"`
	Schedule              []ProfileSchedule      `json:"schedule"`
	Frequencies           []FrequencyEntry       `json:"frequencies"`
	ProfileStates         []ProfileRuntimeStatus `json:"profile_states"`
	Archive               []ArchivedProfile      `json:"archive"`
	Persons               []Person               `json:"persons"`
	ManualRuns            []ManualRun            `json:"manual_runs"`
	TodayRemainingSeconds uint32                 `json:"today_remaining_seconds"`
	Meta                  AppMeta                `json:"meta"`
}

type PersonMutationResult struct {
	Snapshot Snapshot `json:"snapshot"`
	PersonID string   `json:"person_id"`
}

type AIContextRequest struct {
	PersonID  string   `json:"person_id"`
	PersonIDs []string `json:"person_ids"`
	Mode      string   `json:"mode"`
}

type AIImportPreview struct {
	PersonID       string   `json:"person_id"`
	PersonName     string   `json:"person_name"`
	PhaseCount     int      `json:"phase_count"`
	ProgramCount   int      `json:"program_count"`
	TotalDays      int      `json:"total_days"`
	ReplacesActive bool     `json:"replaces_active"`
	Summary        []string `json:"summary"`
	Warnings       []string `json:"warnings,omitempty"`
	// UnknownFields to pełne ścieżki pól, których parser nie zna i zignorował.
	// Import jest nadal możliwy — to ostrzeżenie, nie błąd.
	UnknownFields []string `json:"unknown_fields,omitempty"`
}

// AIImportPreviewBatch to podgląd importu wielu osób naraz — odpowiedź AI może
// być pojedynczym obiektem albo tablicą obiektów (jedna osoba na element).
type AIImportPreviewBatch struct {
	Persons []AIImportPreview `json:"persons"`
}
