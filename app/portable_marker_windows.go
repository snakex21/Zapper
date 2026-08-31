//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

func preparePortableInstall(appDirectory string) {
	markerPath := filepath.Join(appDirectory, portableMarker)
	info, err := os.Stat(markerPath)
	if err != nil || info.IsDir() {
		return
	}
	path, err := syscall.UTF16PtrFromString(markerPath)
	if err != nil {
		return
	}
	attributes, err := syscall.GetFileAttributes(path)
	if err != nil {
		return
	}
	_ = syscall.SetFileAttributes(path, attributes|syscall.FILE_ATTRIBUTE_HIDDEN)

	for _, obsoleteFile := range []string{
		"README_PORTABLE.txt",
		"Zapper.ico",
		filepath.Join("firmware", "languages.json"),
		filepath.Join("firmware", "LANGUAGES.md"),
	} {
		_ = os.Remove(filepath.Join(appDirectory, obsoleteFile))
	}
	for _, obsoleteDirectory := range []string{
		filepath.Join("firmware", "archive"),
		filepath.Join("firmware", "zapper_v5"),
	} {
		_ = os.RemoveAll(filepath.Join(appDirectory, obsoleteDirectory))
	}
}
