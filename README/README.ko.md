**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

새 버전의 애플리케이션은 하나의 창에서 실행되며 Python, Node.js 또는 Wails가 필요하지 않습니다. 보드를 연결하지 않은 상태에서 계획 도구와 로그로 사용할 수도 있고, USB를 통해 Arduino Nano를 제어할 수도 있습니다.

## 라이선스와 책임

코드, firmware, 회로도 및 문서는 **PolyForm Noncommercial 1.0.0** 라이선스에 따라 비상업적 용도로 공개되어 있습니다. 해당 라이선스가 허용하는 범위에서 사용, 연구, 수정 및 배포할 수 있지만, 저자의 별도 허가 없이 프로젝트를 상업적으로 사용할 수 없습니다. 자세한 내용은 `LICENSE` 파일을 확인하십시오.

이 프로젝트는 개인 실험과 DIY 사용을 위해 보증 없이 제공됩니다. 올바른 조립, 수정 및 장치 사용 방식에 대한 책임은 사용자에게 있습니다. 저자는 하드웨어 손상, 기타 손실 또는 잘못된 조립이나 사용으로 인한 결과에 대해 책임을 지지 않으며 특정한 건강 효과를 보장하지 않습니다.

## 애플리케이션 실행

portable 버전 폴더에서 `Zapper.exe`를 실행합니다. 고정된 사람과 식별자는 `data/persons.json`, 활성 프로필은 `data/profiles.json`에 저장되며, 각 실행은 `data/progress/`에 별도의 파일을 가집니다. 완료된 실행은 `profile.json`과 `progress.json`이 들어 있는 `data/archive/<id>/` 폴더로 이동됩니다. 보드 설정은 `data/device.json`, 감지되거나 선택된 언어를 포함한 애플리케이션 설정은 `data/settings.json`에 저장됩니다. 백업은 로컬 `backups/` 하위 폴더에 남습니다. 모든 데이터는 EXE 옆에 있으며 AppData, Documents 또는 Windows 레지스트리에는 아무것도 기록하지 않습니다.

**프로필** 화면에서는 사람을 추가하고, 클립보드에 복사할 수 있는 AI용 컨텍스트 텍스트를 생성하며, AI 모델이 반환한 단순화된 JSON을 붙여넣을 수 있습니다. 이 형식에서 주파수는 `frequency_hz`로 지정합니다. 애플리케이션은 프로필을 검증하고 미리보기를 표시한 뒤 확인 후에만 새 `run_id`를 만듭니다. 해당 사람의 이전 활성 실행은 먼저 보관됩니다.

프로필 세션 중 **일시정지** 버튼은 현재 단계의 남은 부분과 이후 모든 단계를 로컬 진행 상태에 저장합니다. 다시 시작하면 변경되지 않은 firmware로 짧아진 시퀀스를 보내고 보드에서 다시 물리적 확인을 요구합니다. **중지**는 부분 진행을 취소하고 전체 세션을 다시 실행할 수 있는 상태로 남깁니다.

건너뛴 세션은 기한이 지난 항목으로 대기열에 남습니다. 프로그램 규칙은 파트 수, 시리즈 내부의 휴식, 전체 세션 사이의 간격, 세션 후 쿨다운, 같은 날 다른 프로그램과의 호환성을 정의합니다. 기한이 지난 세션이 없는 프로필은 계획이 완료되면 자동으로 보관되며, **프로그램 종료**를 사용하면 더 일찍 닫을 수 있습니다.

## 애플리케이션 언어

시작 시 애플리케이션은 Windows/WebView2의 언어를 읽어 지원되는 30개 언어 중 하나에 대응시킵니다. 설정이 **자동(Windows)** 모드인 동안에는 시작할 때마다 언어를 감지합니다. 왼쪽 패널에서 수동으로 선택한 언어는 `data/settings.json`에 저장되며 자동 모드를 다시 선택할 때까지 자동 변경을 중지합니다.

애플리케이션 언어는 firmware 변형의 기본 언어이기도 합니다. 표준 LCD1602/HD44780에서 안정적으로 표시할 수 없는 문자 체계의 경우 애플리케이션은 해당 firmware 변형을 선택하되 LCD 텍스트는 영어를 사용합니다. 데스크톱 인터페이스는 계속 선택한 언어를 사용합니다.

## Arduino와 USB

현재 firmware는 `firmware/zapper_v5/zapper_v5.ino`에 있으며 설명은 `firmware/zapper_v5/README.md`에 있습니다. firmware를 업로드한 뒤에는 다음 순서로 사용합니다.

1. **장치** 화면을 엽니다.
2. COM 포트를 선택하고 **연결**을 클릭합니다.
3. **준비됨** 상태가 될 때까지 기다립니다.
4. 오늘의 세션을 보내거나 수동 모드에서 단일 값을 시작합니다.
5. 보드의 연결을 확인한 뒤 물리 버튼을 누릅니다. 출력은 그 후에 시작됩니다.

선택한 포트는 로컬 `data/device.json` 파일에 기억됩니다. 프로필 세션은 별도의 정확한 `device_steps`를 저장합니다. “30 kHz” 같은 설명은 사람이 읽기 위한 텍스트로 유지되고, 보드에는 `30000000` 밀리헤르츠와 밀리초 단위의 시간이 전달됩니다.

### LCD firmware 언어

Firmware 5.1.0에는 하나의 코드베이스에서 만들어지는 30개의 독립 언어 변형이 있습니다. 각 Arduino sketch에는 LCD 텍스트 한 세트만 들어 있습니다. 라틴 알파벳을 사용하는 언어는 안전한 ASCII로 저장된 짧은 전용 문구를 사용합니다. 키릴 문자 등 일반적인 LCD1602/HD44780에서 안정적으로 표시할 수 없는 문자 체계의 경우 해당 변형은 영어 LCD 인터페이스를 사용합니다. 전체 목록은 `firmware/LANGUAGES.md`에 있습니다.

`go run ./tools/firmware_i18n` 명령은 모든 sketch를 `build/generated/firmware/`에 생성합니다. 일반적인 `build.ps1` 과정에서는 이를 자동으로 수행하고 각 변형을 portable 버전에 포함합니다.

### 애플리케이션에서 firmware 업로드

**장치 → Firmware** 영역은 감지된 버전, 최신 버전, firmware 변형의 언어와 LCD 언어를 표시합니다. 사용자는 Arduino Nano의 새 bootloader 또는 이전 bootloader를 선택한 뒤 명시적으로 **Firmware 업로드**를 클릭합니다. 애플리케이션이 시작될 때 자동으로 보드에 firmware를 기록하지 않습니다.

컴파일과 업로드는 `arduino-cli`가 담당합니다. Zapper는 `tools/arduino-cli/`, EXE 옆, `PATH`, 일반적인 Arduino IDE 설치 위치에서 이를 찾습니다. 도구가 없으면 애플리케이션이 이를 명확히 표시하고 업로드 버튼은 비활성 상태로 유지됩니다. 컴파일하려면 사용하는 `arduino-cli` 설치에 `arduino:avr` core와 `LiquidCrystal_I2C` 라이브러리도 있어야 합니다.

### 언어 감지와 firmware 선택

시작 시 애플리케이션은 WebView2/Windows 환경 언어(`navigator.languages`)를 읽어 지원되는 30개 코드 중 하나에 대응시킵니다. 시스템 언어가 지원되지 않으면 영어를 선택합니다. **자동(Windows)** 모드에서는 시작할 때마다 언어를 확인하며, 수동 선택은 자동 모드를 다시 활성화할 때까지 `data/settings.json`에 저장됩니다.

같은 언어 코드가 firmware 업로드 화면의 기본 선택이 됩니다. LCD1602에서 지원하지 않는 언어의 경우에도 애플리케이션은 사용자 언어로 표시된 변형을 선택하지만 LCD 텍스트가 영어라는 점을 알려 줍니다. 애플리케이션 시작 시 firmware가 자동으로 업로드되는 일은 없습니다. Arduino에 이미 저장된 다른 프로그램을 실수로 덮어쓰지 않도록 업로드에는 사용자의 명시적인 클릭이 필요합니다.

## 빌드

Go가 필요합니다. 가장 간단한 방법은 프로젝트 루트 폴더에서 다음을 실행하는 것입니다.

```text
build.bat
```

또는 PowerShell에서 다음을 실행할 수 있습니다.

```powershell
.\build.ps1
```

스크립트는 테스트와 코드 분석을 실행하고 `build/generated/Zapper-dev.exe`를 빌드한 뒤, 콘솔 창이 없는 portable `build/Zapper/Zapper.exe`를 준비합니다.

## 프로젝트 구조

- `app/` — Go 코드, HTML/CSS/JS 인터페이스, 가이드, 주파수 데이터베이스.
- `firmware/zapper_v5/` — 현재 Arduino firmware.
- `data/` — 활성 프로필, 진행 상태, 보관 자료, 장치 설정, 자동 백업.
- `locales/` — 버전 관리되는 UI 및 안내서 번역으로, 개발 시 사용되고 릴리스에 복사됩니다.
- `build/Zapper/` — 다른 컴퓨터로 복사할 수 있는 완성된 portable 버전.