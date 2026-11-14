package main

import (
	"strings"
	"unicode"
)

// defaultWindowTitle jest neutralny językowo: okno startuje ukryte, a i18n.js
// ustawia przetłumaczony tytuł (apiSetWindowTitle), zanim okno zostanie pokazane.
const defaultWindowTitle = "Zapper"

const maxWindowTitleRunes = 120

// nativeWindowTitle czyści tytuł przychodzący z interfejsu WWW: usuwa znaki
// sterujące, przycina długość i nigdy nie zwraca pustego tytułu.
func nativeWindowTitle(title string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, title)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if runes := []rune(cleaned); len(runes) > maxWindowTitleRunes {
		cleaned = strings.TrimSpace(string(runes[:maxWindowTitleRunes]))
	}
	if cleaned == "" {
		return defaultWindowTitle
	}
	return cleaned
}
