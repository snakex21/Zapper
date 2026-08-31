//go:build !windows

package main

import "unsafe"

type nativeStartupOverlay struct{}

func setNativeWindowVisible(window unsafe.Pointer, visible bool) {}

func prepareNativeApplicationWindow(window unsafe.Pointer, maximized bool) {}

func revealNativeApplicationWindow(window unsafe.Pointer) {}

func createNativeStartupOverlay(parent unsafe.Pointer, previewPNG []byte) (*nativeStartupOverlay, error) {
	return nil, nil
}

func destroyNativeStartupOverlay(overlay *nativeStartupOverlay) {}

func setNativeWindowMaximized(window unsafe.Pointer, maximized bool) {}

func setNativeWindowIcon(window unsafe.Pointer) {}

func nativeWindowState(window unsafe.Pointer) (windowState, bool) {
	return windowState{}, false
}
