package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	webview2 "github.com/jchv/go-webview2"
)

// backgroundErrorSink raportuje błędy powstałe POZA wywołaniem z frontu —
// głównie w callbackach urządzenia, gdzie nie ma komu zwrócić error.
//
// Aplikacja jest budowana z -H windowsgui, więc stderr jest niewidoczny.
// Dlatego każdy błąd trafia do pliku data/errors.log ORAZ, gdy okno już istnieje,
// do tego samego czerwonego banera co błędy JS (window.zapperReportError).
// Użytkownik MUSI się dowiedzieć, że zabieg się nie zapisał.
type backgroundErrorSink struct {
	mu      sync.Mutex
	logPath string
	window  webview2.WebView
	pending []string
}

func newBackgroundErrorSink(logPath string) *backgroundErrorSink {
	return &backgroundErrorSink{logPath: logPath}
}

// attach podpina okno i wypycha błędy zgłoszone, zanim okno powstało.
func (s *backgroundErrorSink) attach(window webview2.WebView) {
	s.mu.Lock()
	s.window = window
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()
	for _, message := range pending {
		s.pushToWindow(window, message)
	}
}

func (s *backgroundErrorSink) report(context string, err error) {
	message := fmt.Sprintf("%s: %v", context, err)
	log.Printf("BŁĄD W TLE — %s", message)
	s.appendToFile(message)

	s.mu.Lock()
	window := s.window
	if window == nil {
		s.pending = append(s.pending, message)
	}
	s.mu.Unlock()
	if window != nil {
		s.pushToWindow(window, message)
	}
}

func (s *backgroundErrorSink) appendToFile(message string) {
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), message)
	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line)
}

// pushToWindow woła istniejący w froncie mechanizm raportowania (czerwony baner).
// Eval musi lecieć przez Dispatch, bo callback urządzenia biegnie w innym wątku
// niż pętla komunikatów okna.
func (s *backgroundErrorSink) pushToWindow(window webview2.WebView, message string) {
	encoded, err := json.Marshal(message)
	if err != nil {
		return
	}
	script := fmt.Sprintf(
		"if (window.zapperReportError) { window.zapperReportError('błąd w tle aplikacji', %s); }",
		string(encoded))
	window.Dispatch(func() {
		window.Eval(script)
	})
}
