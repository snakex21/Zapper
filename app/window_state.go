package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

type windowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

func loadWindowState(appDirectory string) windowState {
	state := windowState{Width: 1380, Height: 860}
	data, err := os.ReadFile(filepath.Join(appDirectory, "data", "window.json"))
	if err != nil {
		return state
	}
	var saved windowState
	if json.Unmarshal(data, &saved) != nil {
		return state
	}
	if saved.Width >= 1060 {
		state.Width = saved.Width
	}
	if saved.Height >= 680 {
		state.Height = saved.Height
	}
	state.Maximized = saved.Maximized
	return state
}

func saveWindowState(appDirectory string, state windowState) error {
	if state.Width < 1060 || state.Height < 680 {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(appDirectory, "data"), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDirectory, "data", "window.json"), data, 0o644)
}

func rememberWindowState(appDirectory string, window unsafe.Pointer, stop <-chan struct{}) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	var last windowState
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			state, ok := nativeWindowState(window)
			if !ok || state == last {
				continue
			}
			_ = saveWindowState(appDirectory, state)
			last = state
		}
	}
}
