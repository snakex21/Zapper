@echo off
setlocal
cd /d "%~dp0"

echo Budowanie Zapper i wersji portable...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0build.ps1"
if errorlevel 1 (
    echo.
    echo BLAD: budowanie nie powiodlo sie.
    pause
    exit /b 1
)

echo.
echo Glowne EXE:         %~dp0Zapper.exe
echo Build developerski: %~dp0build\generated\Zapper-dev.exe
echo Portable:           %~dp0build\Zapper\
pause
