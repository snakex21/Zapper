package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var repetitionPattern = regexp.MustCompile(`(?i)(\d+)\s*x|x\s*(\d+)|(\d+)\s+(?:raz|razy|sesj)`)

var weekdayKeys = map[time.Weekday]string{
	time.Monday:    "Monday",
	time.Tuesday:   "Tuesday",
	time.Wednesday: "Wednesday",
	time.Thursday:  "Thursday",
	time.Friday:    "Friday",
	time.Saturday:  "Saturday",
	time.Sunday:    "Sunday",
}

var weekdayNames = map[time.Weekday]string{
	time.Monday:    "Poniedziałek",
	time.Tuesday:   "Wtorek",
	time.Wednesday: "Środa",
	time.Thursday:  "Czwartek",
	time.Friday:    "Piątek",
	time.Saturday:  "Sobota",
	time.Sunday:    "Niedziela",
}

func parseDate(value string, fallback time.Time) time.Time {
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return dateOnly(fallback)
	}
	return parsed
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

// daysBetween liczy różnicę KALENDARZOWĄ w dniach między dwiema datami.
//
// Nie wolno tu używać from.Sub(to) — time.Sub zwraca czas absolutny, więc doba
// przejścia na czas letni ma 23 h, a doba powrotu na zimowy 25 h. Dzielenie
// takiej różnicy przez 24 h i obcinanie do int gubi (albo dubluje) całą dobę,
// co przesuwało numerację dni planu i granice faz. Obie daty są tu rzutowane na
// północ UTC (strefa bez DST), więc różnica jest zawsze wielokrotnością 24 h.
func daysBetween(from, to time.Time) int {
	fromUTC := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toUTC.Sub(fromUTC) / (24 * time.Hour))
}

func effectiveStartDate(config Config, progress Progress, now time.Time) time.Time {
	if progress.StartDate != "" {
		return parseDate(progress.StartDate, now)
	}
	return parseDate(config.StartDate, now)
}

func profileStartDate(profile Profile, fallback time.Time) time.Time {
	if strings.TrimSpace(profile.StartDate) != "" {
		return parseDate(profile.StartDate, fallback)
	}
	return dateOnly(fallback)
}

func extractRepetitions(note, duration string) string {
	match := repetitionPattern.FindStringSubmatch(note + " " + duration)
	for index := 1; index < len(match); index++ {
		if match[index] != "" {
			if _, err := strconv.Atoi(match[index]); err == nil {
				return match[index] + "x"
			}
		}
	}
	return "1x"
}

// scheduling.Repetitions jest źródłem prawdy dla planera, widoku Dzisiaj i
// Urządzenia. Harmonogram musi pokazywać dokładnie tę samą liczbę. Tekst z pola
// czasu/notatki zostaje wyłącznie jako zgodność ze starszymi profilami, które nie
// miały jeszcze strukturalnego pola repetitions.
func repetitionsLabel(scheduling SessionScheduling, note, duration string) string {
	if scheduling.Repetitions > 0 {
		return fmt.Sprintf("%dx", scheduling.Repetitions)
	}
	return extractRepetitions(note, duration)
}

func planForDate(profile Profile, currentDate, startDate time.Time) DayPlan {
	currentDate = dateOnly(currentDate)
	startDate = dateOnly(startDate)
	dayIndex := daysBetween(startDate, currentDate)
	plan := DayPlan{
		Date:        currentDate.Format("2006-01-02"),
		PlannedDate: currentDate.Format("2006-01-02"),
		DayName:     weekdayNames[currentDate.Weekday()],
		DayNumber:   dayIndex + 1,
		ProfileID:   profile.ID,
		PersonID:    profilePersonID(profile),
		RunID:       profile.RunID,
		ProfileName: profile.Name,
		Time:        "-",
		Repetitions: "-",
	}

	if dayIndex < 0 {
		plan.Status = "before"
		plan.Program = "Przed rozpoczęciem"
		plan.Note = "Start: " + startDate.Format("02.01.2006")
		return plan
	}

	if len(profile.Phases) == 0 {
		plan.Status = "empty"
		plan.Program = "Brak planu"
		plan.Note = "Dodaj fazę w edytorze profilu"
		return plan
	}

	cumulativeDays := 0
	for _, phase := range profile.Phases {
		duration := phase.DurationDays
		if duration < 1 {
			duration = 7
		}
		if dayIndex < cumulativeDays || dayIndex >= cumulativeDays+duration {
			cumulativeDays += duration
			continue
		}

		plan.PhaseName = phase.Name
		plan.PhaseID = phase.ID
		dayInPhase := dayIndex - cumulativeDays
		if phase.Mode == "interval" {
			gap := phase.IntervalGap
			if gap < 0 {
				gap = 0
			}
			if dayInPhase%(gap+1) != 0 {
				plan.Status = "break"
				plan.Program = "Dzień przerwy"
				plan.Note = "Regeneracja"
				return plan
			}
			plan.Status = "session"
			plan.Program = valueOr(phase.Program, "Brak programu")
			plan.Time = valueOr(phase.Time, "-")
			plan.Note = phase.Note
			plan.DeviceSteps = append([]DeviceStep(nil), phase.DeviceSteps...)
			plan.Scheduling = phase.Scheduling
			plan.Repetitions = repetitionsLabel(plan.Scheduling, phase.Note, phase.Time)
			return plan
		}

		daily, exists := phase.Schedule[weekdayKeys[currentDate.Weekday()]]
		if !exists || strings.TrimSpace(daily.Frequency) == "" {
			plan.Status = "break"
			plan.Program = "Wolny dzień"
			return plan
		}
		plan.Status = "session"
		plan.Program = daily.Frequency
		plan.Time = valueOr(daily.Time, "-")
		plan.Note = daily.Note
		plan.DeviceSteps = append([]DeviceStep(nil), daily.DeviceSteps...)
		plan.Scheduling = daily.Scheduling
		plan.Repetitions = repetitionsLabel(plan.Scheduling, daily.Note, daily.Time)
		return plan
	}

	plan.Status = "complete"
	plan.Program = "Program zakończony"
	plan.Note = "Wszystkie fazy zostały ukończone"
	return plan
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
