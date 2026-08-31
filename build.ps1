$ErrorActionPreference = "Stop"
Set-Location -LiteralPath $PSScriptRoot

$generatedRoot = Join-Path $PSScriptRoot "build\generated"
if (Test-Path -LiteralPath $generatedRoot) {
    Remove-Item -LiteralPath $generatedRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $generatedRoot -Force | Out-Null

Write-Host "Sprawdzanie kodu..."
go test ./...
go vet ./...

Write-Host "Generowanie 30 wariantow jezykowych firmware..."
go run ./tools/firmware_i18n

Write-Host "Audyt 30/30 jezykow..."
python .\tools\check_i18n.py
if ($LASTEXITCODE -ne 0) { throw "Audyt wielojezycznosci nie przeszedl." }

Write-Host "Audyt README 30/30..."
python .\tools\check_readmes.py
if ($LASTEXITCODE -ne 0) { throw "Audyt README nie przeszedl." }

Write-Host "Audyt komunikatow aktualizatora 30/30..."
node .\tools\check_update_i18n.js
if ($LASTEXITCODE -ne 0) { throw "Audyt tlumaczen aktualizatora nie przeszedl." }

$rootExe = Join-Path $PSScriptRoot "Zapper.exe"
Write-Host "Budowanie Zapper.exe w glownym folderze..."
go build -trimpath -ldflags="-s -w -H=windowsgui" -o $rootExe ./app

$devExe = Join-Path $generatedRoot "Zapper-dev.exe"
Write-Host "Budowanie build\generated\Zapper-dev.exe..."
go build -trimpath -ldflags="-s -w -H=windowsgui" -o $devExe ./app

$portableRoot = Join-Path $PSScriptRoot "build\Zapper"
$portableData = Join-Path $portableRoot "data"
$portableFirmware = Join-Path $portableRoot "firmware"
$portableLocales = Join-Path $portableRoot "locales"

Write-Host "Przygotowywanie wersji portable..."
if (Test-Path -LiteralPath $portableRoot) {
    Remove-Item -LiteralPath $portableRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $portableData -Force | Out-Null
New-Item -ItemType Directory -Path $portableFirmware -Force | Out-Null
New-Item -ItemType Directory -Path $portableLocales -Force | Out-Null

Write-Host "Budowanie build\Zapper\Zapper.exe..."
go build -trimpath -ldflags="-s -w -H=windowsgui -X main.appBuildFlavor=portable" -o (Join-Path $portableRoot "Zapper.exe") ./app

# Wersja portable/release ma być czysta. Nigdy nie kopiujemy do niej lokalnych
# profili, historii, logów, portu COM ani innych prywatnych danych z katalogu data.
# Zapper sam utworzy potrzebne pliki przy pierwszym uruchomieniu.
Write-Host "Tworzenie czystego katalogu data dla wersji portable..."

$sourceLocales = Join-Path $PSScriptRoot "locales"
if (-not (Test-Path -LiteralPath $sourceLocales)) {
    throw "Brak pakietow jezykowych: $sourceLocales"
}
Write-Host "Kopiowanie pakietow jezykowych 30/30 do portable..."
Get-ChildItem -LiteralPath $sourceLocales -File | ForEach-Object {
    Copy-Item -LiteralPath $_.FullName -Destination $portableLocales -Force
}

$generatedFirmware = Join-Path $PSScriptRoot "build\generated\firmware"
if (Test-Path -LiteralPath $generatedFirmware) {
    Write-Host "Dolaczanie 30 wariantow jezykowych firmware do portable..."
    Copy-Item -LiteralPath $generatedFirmware -Destination (Join-Path $portableFirmware "localized") -Recurse -Force
}

# `go build ./...` (bez -o) zapisuje binarke pakietu main jako app.exe w biezacym katalogu.
# Taki artefakt myli przy uruchamianiu, wiec sprzatamy go po kazdym budowaniu.
foreach ($stray in @("$PSScriptRoot\zapper_go.exe", "$PSScriptRoot\app.exe", "$PSScriptRoot\app\app.exe", "$PSScriptRoot\app\zapper_go.exe", "$PSScriptRoot\zapper_gui_test.exe")) {
    if (Test-Path -LiteralPath $stray) {
        Write-Host "Usuwanie przestarzalego artefaktu: $stray"
        Remove-Item -LiteralPath $stray -Force
    }
}

Write-Host "Glowne EXE: $rootExe"
Write-Host "Build developerski: $devExe"
Write-Host "Portable: $portableRoot"
