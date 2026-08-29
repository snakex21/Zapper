@echo off
setlocal
cd /d "%~dp0"

echo Przygotowywanie paczki release Zapper...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0release.ps1"
if errorlevel 1 (
    echo.
    echo BLAD: przygotowanie release nie powiodlo sie.
    pause
    exit /b 1
)

echo.
echo Gotowe pliki release sa w: %~dp0dist\
pause
