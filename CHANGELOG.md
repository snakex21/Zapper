# Changelog

All notable public changes to Zapper are documented here.

## 10.3.0 — 2026-08-29

### Application

- Reworked the desktop application around Go + WebView2 in a single native window.
- Added persistent people, profiles, phases, per-run progress, history and archived runs.
- Added overdue-session handling, multi-part sessions, spacing/cooldown rules and manual completion controls.
- Added pause/resume support that preserves the remaining device steps locally.
- Added direct Arduino Nano USB control, connection recovery and remembered COM-port settings.
- Added in-app firmware information and explicit user-triggered firmware flashing support.
- Added window-size/maximized-state persistence.
- Added a dedicated Zapper application icon for Windows executables, the native window, taskbar, loading screen and in-app branding.

### Firmware

- Current firmware version: 5.1.0.
- Added 30 generated firmware language variants from one maintained firmware source.
- Added LCD English fallback for scripts that a standard LCD1602/HD44780 cannot display portably.
- Kept the archived v4 firmware source alongside the current v5 source.

### Languages and documentation

- Added 30 application languages.
- Added complete UI locale packs, guide translations and localized frequency catalogs.
- Added a strict 30/30 localization audit to the build.
- Added 30 matching README files: English at the repository root and 29 translations in `README/`.
- Added a structural README audit so language versions keep the same layout and technical references.

### Distribution

- Added automatic GitHub update checks. When a newer release is available, the app asks before downloading anything.
- Portable builds can download the matching GitHub release ZIP, verify its SHA-256 checksum, install it without touching `data/`, and restart Zapper automatically.
- Clicking the version number in the sidebar performs a manual update check.
- Portable builds contain a clean `data/` directory and never copy local profiles, history, logs or device settings.
- Added `LICENSE` and `README_PORTABLE.txt` to the portable package.
- Build artifacts are separated from repository source files under `build/` and are ignored by Git.
- Added `release.ps1` / `release.bat` to create a versioned Windows x64 ZIP and SHA-256 checksum in `dist/`.
- Added GitHub Actions validation for pushes/pull requests and automatic GitHub Release publishing for `v*` tags.
