//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func preparePortableInstall(appDirectory string) {
	if appBuildFlavor != "portable" {
		return
	}

	for _, obsoleteFile := range []string{
		".zapper-portable",
		"LICENSE",
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
