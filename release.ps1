param(
    [string]$Tag = ""
)

$ErrorActionPreference = "Stop"
Set-Location -LiteralPath $PSScriptRoot

$appSource = Get-Content -LiteralPath (Join-Path $PSScriptRoot "app\app.go") -Raw
$match = [regex]::Match($appSource, 'const\s+appVersion\s*=\s*"([^"]+)"')
if (-not $match.Success) {
    throw "Nie udało się odczytać wersji aplikacji z app/app.go."
}
$version = $match.Groups[1].Value

if ($Tag) {
    $tagVersion = $Tag.Trim()
    if ($tagVersion.StartsWith("v")) { $tagVersion = $tagVersion.Substring(1) }
    if ($tagVersion -ne $version) {
        throw "Tag $Tag nie zgadza się z appVersion $version."
    }
}

Write-Host "Budowanie release Zapper v$version..."
& (Join-Path $PSScriptRoot "build.ps1")
if ($LASTEXITCODE -ne 0) {
    throw "Build release nie przeszedł."
}

$portableRoot = Join-Path $PSScriptRoot "build\Zapper"
$dataRoot = Join-Path $portableRoot "data"
if (-not (Test-Path -LiteralPath (Join-Path $portableRoot "Zapper.exe"))) {
    throw "Brak build\Zapper\Zapper.exe."
}
if (-not (Test-Path -LiteralPath (Join-Path $portableRoot "LICENSE"))) {
    throw "Brak LICENSE w paczce portable."
}
if (-not (Test-Path -LiteralPath (Join-Path $portableRoot ".zapper-portable"))) {
    throw "Brak znacznika wersji portable wymaganego przez automatyczne aktualizacje."
}
$portableLocalizedFirmware = Join-Path $portableRoot "firmware\localized"
if (-not (Test-Path -LiteralPath $portableLocalizedFirmware)) {
    throw "Brak wariantow firmware w paczce portable."
}
if ((Get-ChildItem -LiteralPath $portableLocalizedFirmware -Directory | Measure-Object).Count -ne 30) {
    throw "Paczka portable nie zawiera dokladnie 30 wariantow firmware."
}
foreach ($developerOnlyPath in @("firmware\archive", "firmware\zapper_v5", "firmware\languages.json", "firmware\LANGUAGES.md")) {
    if (Test-Path -LiteralPath (Join-Path $portableRoot $developerOnlyPath)) {
        throw "Paczka portable zawiera zbedny element deweloperski: $developerOnlyPath"
    }
}
if ((Get-ChildItem -LiteralPath $dataRoot -Force | Measure-Object).Count -ne 0) {
    throw "Katalog data w paczce release nie jest pusty. Release przerwany."
}

$distRoot = Join-Path $PSScriptRoot "dist"
if (Test-Path -LiteralPath $distRoot) {
    Remove-Item -LiteralPath $distRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $distRoot -Force | Out-Null

$packageName = "Zapper-v$version-Windows-x64"
$stagingRoot = Join-Path $distRoot $packageName
Copy-Item -LiteralPath $portableRoot -Destination $stagingRoot -Recurse -Force

$zipPath = Join-Path $distRoot ($packageName + ".zip")
Compress-Archive -LiteralPath $stagingRoot -DestinationPath $zipPath -CompressionLevel Optimal -Force
Remove-Item -LiteralPath $stagingRoot -Recurse -Force

$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath).Hash.ToLowerInvariant()
$hashPath = $zipPath + ".sha256"
[IO.File]::WriteAllText($hashPath, "$hash  $([IO.Path]::GetFileName($zipPath))`r`n", (New-Object System.Text.UTF8Encoding($false)))

Write-Host "Release gotowy:"
Write-Host "  $zipPath"
Write-Host "  $hashPath"
