**Languages:** [English](README.md) · [Polski](README/README.pl.md) · [Deutsch](README/README.de.md) · [Français](README/README.fr.md) · [Español](README/README.es.md) · [Italiano](README/README.it.md) · [Português](README/README.pt.md) · [Nederlands](README/README.nl.md) · [Svenska](README/README.sv.md) · [Norsk](README/README.no.md) · [Dansk](README/README.da.md) · [Suomi](README/README.fi.md) · [Čeština](README/README.cs.md) · [Slovenčina](README/README.sk.md) · [Magyar](README/README.hu.md) · [Română](README/README.ro.md) · [Türkçe](README/README.tr.md) · [Bahasa Indonesia](README/README.id.md) · [Bahasa Melayu](README/README.ms.md) · [Tiếng Việt](README/README.vi.md) · [Русский](README/README.ru.md) · [Українська](README/README.uk.md) · [Български](README/README.bg.md) · [Ελληνικά](README/README.el.md) · [العربية](README/README.ar.md) · [עברית](README/README.he.md) · [हिन्दी](README/README.hi.md) · [中文](README/README.zh.md) · [日本語](README/README.ja.md) · [한국어](README/README.ko.md)

# Zapper — Go + WebView2

The new version of the application runs in a single window and does not require Python, Node.js, or Wails. It can be used as a planner and log without a connected board, or it can control an Arduino Nano over USB.

## License and responsibility

The code, firmware, schematics, and documentation are publicly available for non-commercial use under the **PolyForm Noncommercial 1.0.0** license. They may be used, studied, modified, and distributed for purposes permitted by that license, but the project may not be used commercially without separate permission from the author. See the `LICENSE` file for details.

The project is provided for independent experiments and DIY use without warranty. The user is responsible for correct assembly, modifications, and the way the device is used. The author is not responsible for hardware damage, other losses, or consequences of incorrect assembly or use, and does not guarantee any particular health effects.

## Running the application

Run `Zapper.exe` from the portable release folder. Persistent people and their identifiers are stored in `data/persons.json`, active profiles in `data/profiles.json`, and every run has its own file in `data/progress/`. Completed runs are moved to `data/archive/<id>/` folders containing `profile.json` and `progress.json`. Board settings are stored in `data/device.json`, while application settings, including the detected or selected language, are stored in `data/settings.json`. Backups remain in local `backups/` subfolders. Everything is kept next to the EXE; nothing is written to AppData, Documents, or the Windows Registry.

In the **Profiles** view you can add people, generate ready-to-copy AI context text, and paste simplified JSON returned by an AI model. Frequencies in this format are supplied as `frequency_hz`; the application validates the profile, shows a preview, and creates a new `run_id` only after confirmation. The person's previous active run is archived first.

During a profile session, the **Pause** button saves the remaining part of the current step and all following steps in local progress. Resuming sends a shortened sequence to the unchanged firmware and again requires physical confirmation on the board. **Stop** cancels partial progress and leaves the full session available to run again.

Skipped sessions remain in the queue as overdue. Program rules define the number of parts, breaks inside a series, spacing between full sessions, post-session cooldown, and compatibility with other programs on the same day. A profile with no overdue sessions is archived automatically after the plan is completed, while **Finish program** allows it to be closed earlier.

## Application language

At startup, the application reads the Windows/WebView2 language and maps it to one of 30 supported languages. As long as the setting remains in **Automatic (Windows)** mode, detection is performed at every launch. A manual language choice in the left panel is stored in `data/settings.json` and disables automatic changes until automatic mode is selected again.

The application language is also the default language for the firmware variant. For writing systems that a standard LCD1602/HD44780 cannot display portably, the application selects the matching firmware variant with English LCD text; the desktop interface still uses the selected language.

## Arduino and USB

The current firmware is located at `firmware/zapper_v5/zapper_v5.ino`, with its description in `firmware/zapper_v5/README.md`. After flashing the firmware:

1. Open the **Device** view.
2. Select the COM port and click **Connect**.
3. Wait for the **Ready** state.
4. Send today's session or start a single value in manual mode.
5. Check the handles on the board, then press its physical button; only then will the output start.

The selected port is remembered in local `data/device.json`. Profile sessions store separate, exact `device_steps`; a description such as “30 kHz” remains human-readable text, while the board receives `30000000` millihertz and the duration in milliseconds.

### LCD firmware languages

Firmware 5.1.0 has 30 separate language variants generated from one codebase. Each Arduino sketch contains only one set of LCD strings. Languages using the Latin alphabet have their own short strings stored as safe ASCII. For Cyrillic and other scripts that a typical LCD1602/HD44780 cannot display portably, the corresponding variant uses an English LCD interface. The full list is available in `firmware/LANGUAGES.md`.

The command `go run ./tools/firmware_i18n` generates all sketches in `build/generated/firmware/`. The normal `build.ps1` process does this automatically and includes the variants in the portable build.

### Flashing firmware from the application

The **Device → Firmware** section shows the detected version, latest version, firmware variant language, and LCD language. The user selects the new or old Arduino Nano bootloader and explicitly clicks **Flash firmware**; the application never flashes the board automatically at startup.

Compilation and uploading are handled by `arduino-cli`. Zapper looks for it in `tools/arduino-cli/`, next to the EXE, in `PATH`, and in typical Arduino IDE locations. If the tool is not available, the application states this clearly and the flash button remains disabled. Compilation also requires the `arduino:avr` core and the `LiquidCrystal_I2C` library to be available to the selected `arduino-cli` installation.

### Language detection and firmware selection

At startup, the application reads the WebView2/Windows environment language (`navigator.languages`) and maps it to one of the 30 supported codes. If the system language is not supported, English is selected. In **Automatic (Windows)** mode the language is checked at every launch; a manual selection is stored in `data/settings.json` until automatic mode is enabled again.

The same language code is the default choice for the firmware flashing screen. For languages not supported by LCD1602, the application still selects the variant identified by the user's language, but informs the user that LCD text will be in English. Firmware is never flashed automatically when the application starts; flashing requires an explicit user click so another program already stored on the Arduino is not overwritten accidentally.

## Building

Go is required. The easiest option is to run this in the project root:

```text
build.bat
```

Alternatively, in PowerShell:

```powershell
.\build.ps1
```

The script runs tests, code analysis, builds `build/generated/Zapper-dev.exe`, and prepares the portable `build/Zapper/Zapper.exe` without a console window.

## Project layout

- `app/` — Go code, HTML/CSS/JS interface, guide, and frequency database.
- `firmware/zapper_v5/` — current Arduino firmware.
- `data/` — active profiles, progress, archive, device settings, and automatic backups.
- `locales/` — committed UI and guide translations used by development builds and copied into releases.
- `build/Zapper/` — ready-to-copy portable version for another computer.