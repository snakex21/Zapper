//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image/png"
	"syscall"
	"unsafe"
)

const (
	swHide        = 0
	swShow        = 5
	swMaximize    = 3
	wmSetIcon     = 0x0080
	iconSmall     = 0
	iconBig       = 1
	imageIcon     = 1
	imageBitmap   = 0
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	ssBitmap      = 0x0000000E
	stmSetImage   = 0x0172
	dibRGBColors  = 0
	biRGB         = 0
	swpShowWindow = 0x0040
	dwmwaCloak    = 13
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

type nativeBitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type nativeStartupOverlay struct {
	Window uintptr
	Bitmap uintptr
}

var user32DLL = syscall.NewLazyDLL("user32.dll")
var showWindowProc = user32DLL.NewProc("ShowWindow")
var updateWindowProc = user32DLL.NewProc("UpdateWindow")
var setForegroundWindowProc = user32DLL.NewProc("SetForegroundWindow")
var createWindowExWProc = user32DLL.NewProc("CreateWindowExW")
var destroyWindowProc = user32DLL.NewProc("DestroyWindow")
var setWindowPosProc = user32DLL.NewProc("SetWindowPos")
var getClientRectProc = user32DLL.NewProc("GetClientRect")
var getWindowPlacementProc = user32DLL.NewProc("GetWindowPlacement")
var loadImageWProc = user32DLL.NewProc("LoadImageW")
var sendMessageWProc = user32DLL.NewProc("SendMessageW")
var gdi32DLL = syscall.NewLazyDLL("gdi32.dll")
var createDIBSectionProc = gdi32DLL.NewProc("CreateDIBSection")
var deleteObjectProc = gdi32DLL.NewProc("DeleteObject")
var kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
var getModuleHandleWProc = kernel32DLL.NewProc("GetModuleHandleW")
var dwmapiDLL = syscall.NewLazyDLL("dwmapi.dll")
var dwmSetWindowAttributeProc = dwmapiDLL.NewProc("DwmSetWindowAttribute")

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

func prepareNativeApplicationWindow(window unsafe.Pointer, maximized bool) {
	if window == nil {
		return
	}
	handle := uintptr(window)
	cloaked := uint32(1)
	_, _, _ = dwmSetWindowAttributeProc.Call(handle, uintptr(dwmwaCloak), uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	command := uintptr(swShow)
	if maximized {
		command = uintptr(swMaximize)
	}
	_, _, _ = showWindowProc.Call(handle, command)
	_, _, _ = updateWindowProc.Call(handle)
}

func revealNativeApplicationWindow(window unsafe.Pointer) {
	if window == nil {
		return
	}
	handle := uintptr(window)
	cloaked := uint32(0)
	_, _, _ = dwmSetWindowAttributeProc.Call(handle, uintptr(dwmwaCloak), uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	_, _, _ = updateWindowProc.Call(handle)
	_, _, _ = setForegroundWindowProc.Call(handle)
}

func createNativeStartupOverlay(parent unsafe.Pointer, previewPNG []byte) (*nativeStartupOverlay, error) {
	if parent == nil || len(previewPNG) == 0 {
		return nil, fmt.Errorf("brak obrazu pierwszej klatki")
	}
	preview, err := png.Decode(bytes.NewReader(previewPNG))
	if err != nil {
		return nil, fmt.Errorf("dekodowanie obrazu pierwszej klatki: %w", err)
	}
	bounds := preview.Bounds()
	sourceWidth := bounds.Dx()
	sourceHeight := bounds.Dy()
	client := nativeRect{}
	ok, _, _ := getClientRectProc.Call(uintptr(parent), uintptr(unsafe.Pointer(&client)))
	width := int(client.Right - client.Left)
	height := int(client.Bottom - client.Top)
	if ok == 0 || sourceWidth <= 0 || sourceHeight <= 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("nieprawidłowy rozmiar obrazu pierwszej klatki")
	}
	header := nativeBitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(nativeBitmapInfoHeader{})),
		Width:       int32(width),
		Height:      -int32(height),
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
		SizeImage:   uint32(width * height * 4),
	}
	var pixelMemory unsafe.Pointer
	bitmap, _, _ := createDIBSectionProc.Call(0, uintptr(unsafe.Pointer(&header)), uintptr(dibRGBColors), uintptr(unsafe.Pointer(&pixelMemory)), 0, 0)
	if bitmap == 0 || pixelMemory == nil {
		return nil, fmt.Errorf("nie utworzono bitmapy pierwszej klatki")
	}
	pixels := unsafe.Slice((*byte)(pixelMemory), width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sourceX := x * sourceWidth / width
			sourceY := y * sourceHeight / height
			red, green, blue, _ := preview.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY).RGBA()
			offset := (y*width + x) * 4
			pixels[offset] = byte(blue >> 8)
			pixels[offset+1] = byte(green >> 8)
			pixels[offset+2] = byte(red >> 8)
			pixels[offset+3] = 255
		}
	}
	className, _ := syscall.UTF16PtrFromString("STATIC")
	overlay, _, _ := createWindowExWProc.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(wsChild|wsVisible|ssBitmap),
		0,
		0,
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		0,
		0,
		0,
	)
	if overlay == 0 {
		_, _, _ = deleteObjectProc.Call(bitmap)
		return nil, fmt.Errorf("nie utworzono nakładki pierwszej klatki")
	}
	_, _, _ = sendMessageWProc.Call(overlay, uintptr(stmSetImage), uintptr(imageBitmap), bitmap)
	_, _, _ = setWindowPosProc.Call(overlay, 0, 0, 0, uintptr(width), uintptr(height), uintptr(swpShowWindow))
	_, _, _ = updateWindowProc.Call(overlay)
	return &nativeStartupOverlay{Window: overlay, Bitmap: bitmap}, nil
}

func destroyNativeStartupOverlay(overlay *nativeStartupOverlay) {
	if overlay == nil {
		return
	}
	if overlay.Window != 0 {
		_, _, _ = destroyWindowProc.Call(overlay.Window)
		overlay.Window = 0
	}
	if overlay.Bitmap != 0 {
		_, _, _ = deleteObjectProc.Call(overlay.Bitmap)
		overlay.Bitmap = 0
	}
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
