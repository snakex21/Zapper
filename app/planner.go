package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxTrackedDays = 3660

func repetitionsFor(plan DayPlan) int {
	value := plan.Scheduling.Repetitions
	if value < 1 {
		parsed, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(plan.Repetitions), "x"))
		if err == nil {
			value = parsed
		}
	}
	if value < 1 {
		return 1
	}
	if value > 12 {
		return 12
	}
	return value
}

func expandScheduledPlan(plan DayPlan) []DayPlan {
	if plan.Status != "session" {
		return nil
	}
	count := repetitionsFor(plan)
	result := make([]DayPlan, 0, count)
	for repeat := 1; repeat <= count; repeat++ {
		occurrence := plan
		occurrence.SessionGroupID = sessionGroupIDFor(occurrence)
		occurrence.RepeatIndex = repeat
		occurrence.RepeatCount = count
		occurrence.Repetitions = fmt.Sprintf("%dx", count)
		occurrence.SessionID = sessionIDFor(occurrence, repeat)
		occurrence.Available = true
		result = append(result, occurrence)
	}
	return result
}

func sessionIDFor(plan DayPlan, repeat int) string {
	return fmt.Sprintf("%s:%d", sessionGroupIDFor(plan), repeat)
}

func sessionGroupIDFor(plan DayPlan) string {
	if strings.TrimSpace(plan.RunID) != "" {
		return fmt.Sprintf("%s:%s:%s:%s", plan.ProfileID, plan.RunID, plan.PlannedDate, plan.PhaseID)
	}
	return fmt.Sprintf("%s:%s:%s", plan.ProfileID, plan.PlannedDate, plan.PhaseID)
}

func completionBelongsToRun(plan DayPlan, completion SessionCompletion) bool {
	if strings.TrimSpace(plan.RunID) == "" {
		return strings.TrimSpace(completion.RunID) == ""
	}
	return completion.RunID == plan.RunID
}

func completionGroupID(completion SessionCompletion) string {
	if strings.TrimSpace(completion.SessionGroupID) != "" {
		return completion.SessionGroupID
	}
	separator := strings.LastIndex(completion.SessionID, ":")
	if separator > 0 {
		return completion.SessionID[:separator]
	}
	return fmt.Sprintf("%s:%s", completion.ProfileID, completion.PlannedDate)
}

func buildOperationalPlans(config Config, progress Progress, now time.Time) ([]DayPlan, []ProfileRuntimeStatus) {
	today := dateOnly(now)
	defaultStartDate := effectiveStartDate(config, progress, today)
	trackingSince := parseDate(progress.TrackingSince, today)
	if trackingSince.After(today) {
		trackingSince = today
	}
	if daysBetween(trackingSince, today) > maxTrackedDays {
		trackingSince = today.AddDate(0, 0, -maxTrackedDays)
	}

	plans := make([]DayPlan, 0)
	states := make([]ProfileRuntimeStatus, 0, len(config.Profiles))
	for _, profile := range config.Profiles {
		startDate := profileStartDate(profile, defaultStartDate)
		profilePlans := make([]DayPlan, 0)
		overdueDates := map[string]bool{}
		for current := trackingSince; !current.After(today); current = current.AddDate(0, 0, 1) {
			base := planForDate(profile, current, startDate)
			for _, occurrence := range expandScheduledPlan(base) {
				// Sesja odpuszczona znika z kolejki operacyjnej, ale NIE staje się wykonaniem.
				if dismissed, exists := progress.DismissedSessions[occurrence.SessionID]; exists && dismissed.RunID == occurrence.RunID {
					continue
				}
				completion, done := progress.Completions[occurrence.SessionID]
				occurrence.Done = done
				if paused, exists := progress.PausedSessions[occurrence.SessionID]; exists && paused.RunID == occurrence.RunID {
					occurrence.Paused = true
					occurrence.PausedRecorded = paused.Recorded
					occurrence.RemainingSeconds = paused.RemainingSeconds
					occurrence.DeviceSteps = append([]DeviceStep(nil), paused.RemainingSteps...)
				}
				if done {
					completedAt, valid := parseTimestamp(completion.CompletedAt)
					if !valid || !dateOnly(completedAt.In(now.Location())).Equal(today) {
						continue
					}
				} else if current.Before(today) {
					occurrence.Overdue = true
					overdueDates[occurrence.PlannedDate] = true
				}
				profilePlans = append(profilePlans, occurrence)
			}
		}

		if len(profilePlans) == 0 {
			profilePlans = append(profilePlans, planForDate(profile, today, startDate))
		}
		extensionDays := len(overdueDates)
		for index := range profilePlans {
			profilePlans[index].ExtensionDays = extensionDays
		}
		plans = append(plans, profilePlans...)
		states = append(states, ProfileRuntimeStatus{
			ProfileID:     profile.ID,
			ProfileName:   profile.Name,
			OverdueCount:  countOverdue(profilePlans),
			ExtensionDays: extensionDays,
		})
	}

	evaluateAvailability(plans, progress.Completions, now)
	annotateSessionOrder(plans)
	sort.SliceStable(plans, func(i, j int) bool {
		if plans[i].Done != plans[j].Done {
			return !plans[i].Done
		}
		if plans[i].Overdue != plans[j].Overdue {
			return plans[i].Overdue
		}
		if plans[i].PlannedDate != plans[j].PlannedDate {
			return plans[i].PlannedDate < plans[j].PlannedDate
		}
		if plans[i].ProfileName != plans[j].ProfileName {
			return plans[i].ProfileName < plans[j].ProfileName
		}
		return plans[i].RepeatIndex < plans[j].RepeatIndex
	})
	return plans, states
}

// applyScheduleProgress nanosi wykonania na dzień harmonogramu. Harmonogram
// pokazuje jeden wiersz na dzień, a wykonania zapisane są per powtórzenie
// sesji — dzień uznajemy za wykonany, gdy wszystkie jego powtórzenia mają wpis
// w Completions z tego samego przebiegu (RunID).
func applyScheduleProgress(plan DayPlan, progress *Progress) DayPlan {
	if plan.Status != "session" || len(progress.Completions) == 0 {
		return plan
	}
	count := repetitionsFor(plan)
	doneCount := 0
	for repeat := 1; repeat <= count; repeat++ {
		completion, exists := progress.Completions[sessionIDFor(plan, repeat)]
		if exists && completion.RunID == plan.RunID {
			doneCount++
		}
	}
	if doneCount > 0 && doneCount < count {
		plan.Note = fmt.Sprintf("Wykonano %d z %d", doneCount, count)
	}
	plan.Done = doneCount == count
	return plan
}

func countOverdue(plans []DayPlan) int {
	groups := map[string]bool{}
	for _, plan := range plans {
		if plan.Overdue && !plan.Done {
			groups[plan.SessionGroupID] = true
		}
	}
	return len(groups)
}

// annotateSessionOrder nie blokuje świadomej zmiany kolejności. Daje jednak UI
// informację, że przed wybraną sesją pozostały starsze terminy tej samej osoby,
// dzięki czemu przypadkowy skok z pozycji 4 na 8 wymaga potwierdzenia.
func annotateSessionOrder(plans []DayPlan) {
	for index := range plans {
		plan := &plans[index]
		if plan.Status != "session" || plan.Done {
			continue
		}
		olderGroups := map[string]bool{}
		for _, candidate := range plans {
			if candidate.Status != "session" || candidate.Done || candidate.ProfileID != plan.ProfileID || candidate.RunID != plan.RunID {
				continue
			}
			if candidate.SessionGroupID == plan.SessionGroupID || candidate.PlannedDate >= plan.PlannedDate {
				continue
			}
			olderGroups[candidate.SessionGroupID] = true
		}
		plan.OlderPendingCount = len(olderGroups)
	}
}

// completionIndex to jednorazowo zbudowany indeks wykonań.
// Wcześniej evaluateAvailability robiło CZTERY pełne przejścia po wszystkich
// wykonaniach na każdy plan (O(plany × wykonania × 4)). Indeks zawęża pracę do
// wykonań jednego profilu, a same warunki liczymy w jednym przebiegu.
// Zmiana jest czysto wydajnościowa — kolejność i treść blokad bez zmian.
type completionIndex struct {
	byProfile map[string][]SessionCompletion
}

func buildCompletionIndex(completions map[string]SessionCompletion) completionIndex {
	index := completionIndex{byProfile: make(map[string][]SessionCompletion, len(completions))}
	for _, completion := range completions {
		index.byProfile[completion.ProfileID] = append(index.byProfile[completion.ProfileID], completion)
	}
	return index
}

func evaluateAvailability(plans []DayPlan, completions map[string]SessionCompletion, now time.Time) {
	index := buildCompletionIndex(completions)
	today := dateOnly(now)
	for planIndex := range plans {
		plan := &plans[planIndex]
		if plan.Status != "session" || plan.Done {
			plan.Available = plan.Done
			continue
		}
		plan.Available = true

		if plan.RepeatIndex > 1 {
			previousID := sessionIDFor(*plan, plan.RepeatIndex-1)
			previous, exists := completions[previousID]
			if !exists {
				blockPlan(plan, fmt.Sprintf("Najpierw wykonaj część %d z %d", plan.RepeatIndex-1, plan.RepeatCount), time.Time{})
				continue
			}
			if completedAt, valid := parseTimestamp(previous.CompletedAt); valid {
				required := time.Duration(plan.Scheduling.BreakBetweenMinutes) * time.Minute
				availableAt := completedAt.Add(required)
				if required > 0 && now.Before(availableAt) {
					blockPlan(plan, fmt.Sprintf("Przerwa po poprzedniej części: %d min", plan.Scheduling.BreakBetweenMinutes), availableAt)
					continue
				}
			}
		}

		// Jeden przebieg po wykonaniach tego profilu zbiera wszystkie trzy warunki.
		latestSameProgram := time.Time{}
		cooldownUntil := time.Time{}
		cooldownMinutes := 0
		incompatibleToday := false
		for _, completion := range index.byProfile[plan.ProfileID] {
			if !completionBelongsToRun(*plan, completion) || completionGroupID(completion) == plan.SessionGroupID {
				continue
			}
			completedAt, valid := parseTimestamp(completion.CompletedAt)
			if !valid {
				continue
			}
			sameProgram := completion.ProgramID == plan.Scheduling.ProgramID
			if sameProgram {
				if completedAt.After(latestSameProgram) {
					latestSameProgram = completedAt
				}
			} else if completion.Scheduling.CooldownAfterMinutes > 0 {
				// Zgodność same_day_with znosi regułę "jeden program dziennie",
				// ale NIE znosi jawnie ustawionego cooldownu. Użytkownik, który
				// wpisał 4 h odstępu, ma dostać 4 h, a nie ciche zero.
				availableAt := completedAt.Add(time.Duration(completion.Scheduling.CooldownAfterMinutes) * time.Minute)
				if availableAt.After(cooldownUntil) {
					cooldownUntil = availableAt
					cooldownMinutes = completion.Scheduling.CooldownAfterMinutes
				}
			}
			if !incompatibleToday &&
				dateOnly(completedAt.In(now.Location())).Equal(today) &&
				!programsCompatible(plan.Scheduling, completion.Scheduling) {
				incompatibleToday = true
			}
		}

		if !cooldownUntil.IsZero() && now.Before(cooldownUntil) {
			blockPlan(plan, fmt.Sprintf("Odstęp po poprzedniej sesji: %d min", cooldownMinutes), cooldownUntil)
			continue
		}

		if !latestSameProgram.IsZero() && plan.Scheduling.MinGapMinutes > 0 {
			availableAt := latestSameProgram.Add(time.Duration(plan.Scheduling.MinGapMinutes) * time.Minute)
			if now.Before(availableAt) {
				blockPlan(plan, fmt.Sprintf("Minimalny odstęp dla tego programu: %d min", plan.Scheduling.MinGapMinutes), availableAt)
				continue
			}
		}

		if incompatibleToday {
			blockPlan(plan, "Ten program nie jest oznaczony jako zgodny z już wykonaną dziś sesją", today.AddDate(0, 0, 1))
		}
	}

	blockLaterOverdueOccurrences(plans)
}

// blockLaterOverdueOccurrences pilnuje kolejności zaległości w obrębie jednej partii planów.
// Bez tego użytkownik z pięcioma zaległymi sesjami tego samego programu widział pięć
// przycisków "dostępne", startował drugą i dostawał odmowę dopiero z urządzenia.
// Dostępna zostaje najstarsza niewykonana sesja danego programu; kolejne dostają
// czytelny powód wskazujący, którą zaległość trzeba wykonać najpierw.
func blockLaterOverdueOccurrences(plans []DayPlan) {
	type earliest struct {
		groupID     string
		plannedDate string
	}
	firstPending := map[string]earliest{}
	for index := range plans {
		plan := &plans[index]
		if plan.Status != "session" || plan.Done || !plan.Available {
			continue
		}
		key := plan.ProfileID + "|" + plan.RunID + "|" + plan.Scheduling.ProgramID + "|" + plan.PhaseID
		previous, exists := firstPending[key]
		if !exists {
			firstPending[key] = earliest{groupID: plan.SessionGroupID, plannedDate: plan.PlannedDate}
			continue
		}
		// Powtórzenia w obrębie tej samej sesji mają własną, dokładniejszą blokadę.
		if previous.groupID == plan.SessionGroupID {
			continue
		}
		blockPlan(plan, fmt.Sprintf("Najpierw wykonaj zaległą sesję z %s", formatPlannedDate(previous.plannedDate)), time.Time{})
	}
}

func formatPlannedDate(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02.01.2006")
}

func programsCompatible(first, second SessionScheduling) bool {
	if first.ProgramID == "" || second.ProgramID == "" {
		return false
	}
	return containsProgram(first.SameDayWith, second.ProgramID) && containsProgram(second.SameDayWith, first.ProgramID)
}

func containsProgram(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(wanted)) {
			return true
		}
	}
	return false
}

func blockPlan(plan *DayPlan, reason string, availableAt time.Time) {
	plan.Available = false
	plan.BlockedReason = reason
	if !availableAt.IsZero() {
		plan.AvailableAt = availableAt.Format(time.RFC3339)
	}
}

func parseTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, value)
	return parsed, err == nil
}
