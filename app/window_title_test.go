package main

import (
	"strings"
	"testing"
)

func TestNativeWindowTitle(t *testing.T) {
	cases := map[string]string{
		"Zapper — Planer und Protokoll": "Zapper — Planer und Protokoll",
		"  Zapper —\nплан\tи журнал  ":  "Zapper — план и журнал",
		"":                       defaultWindowTitle,
		" \r\n\t ":               defaultWindowTitle,
		"Zapper — المخطط والسجل": "Zapper — المخطط والسجل",
	}
	for input, want := range cases {
		if got := nativeWindowTitle(input); got != want {
			t.Errorf("nativeWindowTitle(%q) = %q, want %q", input, got, want)
		}
	}
	long := strings.Repeat("ż", maxWindowTitleRunes+50)
	if got := []rune(nativeWindowTitle(long)); len(got) != maxWindowTitleRunes {
		t.Errorf("long title has %d runes, want %d", len(got), maxWindowTitleRunes)
	}
}
