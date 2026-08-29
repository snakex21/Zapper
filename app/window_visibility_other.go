//go:build !windows

package main

import "unsafe"

func setNativeWindowVisible(window unsafe.Pointer, visible bool) {}

func setNativeWindowMaximized(window unsafe.Pointer, maximized bool) {}

func setNativeWindowIcon(window unsafe.Pointer, iconPath string) {}

func nativeWindowState(window unsafe.Pointer) (windowState, bool) {
	return windowState{}, false
}
