//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	swHide     = 0
	swShow     = 5
	swMaximize = 3
	wmSetIcon  = 0x0080
	iconSmall  = 0
	iconBig    = 1
	imageIcon  = 1
)

type nativePoint struct {
	X int32
	Y int32
}

type nativeRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type nativeWindowPlacement struct {
	Length         uint32
	Flags          uint32
	ShowCmd        uint32
	MinPosition    nativePoint
	MaxPosition    nativePoint
	NormalPosition nativeRect
}

var user32DLL = syscall.NewLazyDLL("user32.dll")
var showWindowProc = user32DLL.NewProc("ShowWindow")
var getWindowPlacementProc = user32DLL.NewProc("GetWindowPlacement")
var loadImageWProc = user32DLL.NewProc("LoadImageW")
var sendMessageWProc = user32DLL.NewProc("SendMessageW")
var kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
var getModuleHandleWProc = kernel32DLL.NewProc("GetModuleHandleW")

func setNativeWindowVisible(window unsafe.Pointer, visible bool) {
	if window == nil {
		return
	}
	command := uintptr(swHide)
	if visible {
		command = uintptr(swShow)
	}
	_, _, _ = showWindowProc.Call(uintptr(window), command)
}

func setNativeWindowMaximized(window unsafe.Pointer, maximized bool) {
	if window == nil || !maximized {
		return
	}
	_, _, _ = showWindowProc.Call(uintptr(window), uintptr(swMaximize))
}

func setNativeWindowIcon(window unsafe.Pointer) {
	if window == nil {
		return
	}
	module, _, _ := getModuleHandleWProc.Call(0)
	if module == 0 {
		return
	}
	// ID 1 to RT_GROUP_ICON osadzony w rsrc_windows_amd64.syso.
	big, _, _ := loadImageWProc.Call(module, 1, uintptr(imageIcon), 32, 32, 0)
	if big != 0 {
		_, _, _ = sendMessageWProc.Call(uintptr(window), uintptr(wmSetIcon), uintptr(iconBig), big)
	}
	small, _, _ := loadImageWProc.Call(module, 1, uintptr(imageIcon), 16, 16, 0)
	if small != 0 {
		_, _, _ = sendMessageWProc.Call(uintptr(window), uintptr(wmSetIcon), uintptr(iconSmall), small)
	}
}

func nativeWindowState(window unsafe.Pointer) (windowState, bool) {
	if window == nil {
		return windowState{}, false
	}
	placement := nativeWindowPlacement{Length: uint32(unsafe.Sizeof(nativeWindowPlacement{}))}
	ok, _, _ := getWindowPlacementProc.Call(uintptr(window), uintptr(unsafe.Pointer(&placement)))
	if ok == 0 {
		return windowState{}, false
	}
	rect := placement.NormalPosition
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return windowState{}, false
	}
	return windowState{Width: width, Height: height, Maximized: placement.ShowCmd == swMaximize}, true
}
