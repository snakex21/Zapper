//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func startUpdateInstaller(appDirectory, payloadRoot, updateRoot string) error {
	if !fileExists(filepath.Join(payloadRoot, "Zapper.exe")) {
		return fmt.Errorf("brak Zapper.exe w pobranej aktualizacji")
	}
	scriptPath := filepath.Join(updateRoot, "apply-update.ps1")
	quote := func(value string) string {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$source = %s
$destination = %s
$updateRoot = %s
$pidToWait = %d

try { Wait-Process -Id $pidToWait -ErrorAction SilentlyContinue } catch {}
Start-Sleep -Milliseconds 500

if (-not (Test-Path -LiteralPath (Join-Path $source 'Zapper.exe'))) {
    throw 'Brak Zapper.exe w paczce aktualizacji.'
}
if (-not (Test-Path -LiteralPath (Join-Path $source '.zapper-portable'))) {
    throw 'Paczka nie jest prawidłowym wydaniem portable.'
}

# Kopiujemy nowe wydanie na istniejącą instalację, ale nie kasujemy danych
# użytkownika ani dodatkowych narzędzi. Nowe pliki zastępują stare, a katalog
# data/ pozostaje dokładnie taki, jaki był przed aktualizacją.
$robocopyArgs = @($source, $destination, '/E', '/R:2', '/W:1', '/XD', (Join-Path $source 'data'), '/NFL', '/NDL', '/NJH', '/NJS', '/NP')
& robocopy.exe @robocopyArgs | Out-Null
$copyExit = $LASTEXITCODE
if ($copyExit -gt 7) {
    throw "Robocopy zakończył się kodem $copyExit."
}

$exe = Join-Path $destination 'Zapper.exe'
Start-Process -FilePath $exe -WorkingDirectory $destination
Start-Sleep -Milliseconds 500
try { Remove-Item -LiteralPath $updateRoot -Recurse -Force -ErrorAction SilentlyContinue } catch {}
`, quote(payloadRoot), quote(appDirectory), quote(updateRoot), os.Getpid())
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("nie udało się przygotować instalatora aktualizacji: %w", err)
	}

	command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("nie udało się uruchomić instalatora aktualizacji: %w", err)
	}
	return nil
}
