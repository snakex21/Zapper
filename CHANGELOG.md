# Changelog

All notable public changes to Zapper are documented here.

## 10.3.7 — 2026-08-31

### Startup

- Fixed the application interface remaining invisible after the loading screen introduced in 10.3.6.
- Restored the normal light application background immediately after the dark startup screen finishes.
- Added a regression test ensuring the startup-only visibility rule is removed when initialization finishes.

## 10.3.6 — 2026-08-31

### Startup

- Removed the visible flash between the native startup screen and the application interface.
- WebView documents receive the dark startup background before their first frame is painted.
- The embedded critical loading styles and logo now exactly match the preceding startup screen, without waiting for an image or external stylesheet.

## 10.3.5 — 2026-08-31

### Distribution

- Removed `.zapper-portable` and `LICENSE` completely from application folders and release ZIP files.
- Portable status is now compiled directly into `Zapper.exe`; update payloads are validated by their required application directory structure.
- Portable startup removes marker and license files left behind by older releases.

## 10.3.4 — 2026-08-31

### Distribution

- Removed the redundant external `Zapper.ico` from portable packages; the native window now loads icon resource ID 1 directly from `Zapper.exe`.
- Kept the updater compatibility marker but marked it as a hidden Windows file, and the application restores that hidden attribute at startup after updates.
- Portable startup removes obsolete package-owned files left behind by non-destructive updates from older releases.

## 10.3.3 — 2026-08-31

### Distribution

- Removed the unnecessary Polish-only `README_PORTABLE.txt` from portable packages.
- Removed the developer-only archived v4 source and duplicate base v5 source from portable packages.
- Portable packages now contain only the 30 firmware variants actually used by the in-app firmware flasher under `firmware/localized`.
- Firmware compilation staging now stays inside the portable application folder under `data/firmware-build`.

## 10.3.2 — 2026-08-31

### Reliability

- Fixed automatic updates waiting forever after download because the WebView window was terminated from the wrong Windows thread.
- The application now closes on the UI thread, lets the updater replace the running executable and starts the updated Zapper automatically.
- Added a 30-second shutdown timeout, automatic recovery launch and a persistent local update log in `data/update.log`.
- Update downloads and staging now stay inside the portable application folder under `data/update-staging` instead of using the Windows profile or temporary directories.
- Added a regression test that requires application termination to be dispatched to the WebView UI thread.

## 10.3.1 — 2026-08-31

### Application

- Added a live countdown for required breaks between treatment parts in both the Today and Device views.
- The countdown uses the locally persisted completion timestamp, so closing and reopening the application does not reset the break.
- Sessions unlock automatically when the countdown reaches zero, without restarting or manually refreshing the application.
- Completing the final treatment part removes the finished program from the active plan and keeps it in the local archive.

### Languages

- The countdown reuses the localized `remaining` label and locale-aware time formatting across all 30 supported application languages.

### Reliability

- Added a restart regression test covering a treatment completed at 19:00, a 20-minute break, automatic availability at 19:20 and final program archival.

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
