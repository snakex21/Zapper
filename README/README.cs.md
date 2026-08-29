**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Nová verze aplikace běží v jednom okně a nevyžaduje Python, Node.js ani Wails. Lze ji používat jako plánovač a deník bez připojené desky nebo k ovládání Arduino Nano přes USB.

## Licence a odpovědnost

Kód, firmware, schémata a dokumentace jsou veřejně dostupné pro nekomerční použití pod licencí **PolyForm Noncommercial 1.0.0**. Je možné je používat, studovat, upravovat a šířit pro účely povolené touto licencí, projekt však nelze používat komerčně bez samostatného souhlasu autora. Podrobnosti jsou v souboru `LICENSE`.

Projekt je poskytován bez záruky pro vlastní experimenty a DIY použití. Uživatel odpovídá za správné sestavení, úpravy a způsob používání zařízení. Autor neodpovídá za poškození hardwaru, jiné škody ani následky nesprávného sestavení nebo použití a nezaručuje žádné konkrétní zdravotní účinky.

## Spuštění aplikace

Spusťte `Zapper.exe` ze složky portable verze. Trvalé osoby a jejich identifikátory jsou uloženy v `data/persons.json`, aktivní profily v `data/profiles.json` a každý běh má vlastní soubor v `data/progress/`. Dokončené běhy se přesouvají do složek `data/archive/<id>/`, které obsahují `profile.json` a `progress.json`. Nastavení desky jsou v `data/device.json`, nastavení aplikace včetně zjištěného nebo vybraného jazyka v `data/settings.json`. Zálohy zůstávají v místních podsložkách `backups/`. Vše je vedle souboru EXE; nic se nezapisuje do AppData, Dokumentů ani registru Windows.

V zobrazení **Profily** lze přidávat osoby, vytvářet připravený kontextový text pro AI do schránky a vkládat zjednodušený JSON vrácený modelem AI. Frekvence se v tomto formátu zadávají jako `frequency_hz`; aplikace profil ověří, zobrazí náhled a nový `run_id` vytvoří až po potvrzení. Předchozí aktivní běh dané osoby se nejprve archivuje.

Během profilové relace tlačítko **Pauza** uloží zbývající část aktuálního kroku a všechny následující kroky do místního průběhu. Při pokračování se do nezměněného firmwaru odešle zkrácená sekvence a znovu je vyžadováno fyzické potvrzení na desce. **Zastavit** zruší částečný průběh a ponechá celou relaci k opětovnému spuštění.

Přeskočené relace zůstávají ve frontě jako zpožděné. Pravidla programu určují počet částí, přestávky uvnitř série, rozestup mezi úplnými relacemi, dobu zotavení po relaci a kompatibilitu s jinými programy ve stejný den. Profil bez zpožděných relací se po dokončení plánu automaticky archivuje, zatímco **Ukončit program** umožňuje jeho dřívější uzavření.

## Jazyk aplikace

Při spuštění aplikace načte jazyk Windows/WebView2 a přiřadí jej k jednomu z 30 podporovaných jazyků. Dokud je nastaven režim **Automaticky (Windows)**, zjištění jazyka se provádí při každém spuštění. Ruční výběr jazyka v levém panelu se uloží do `data/settings.json` a vypne automatické změny, dokud není znovu zvolen automatický režim.

Jazyk aplikace je zároveň výchozím jazykem varianty firmwaru. Pro písma, která standardní LCD1602/HD44780 neumí spolehlivě zobrazit, aplikace vybere odpovídající variantu firmwaru s anglickými texty LCD; desktopové rozhraní nadále používá zvolený jazyk.

## Arduino a USB

Aktuální firmware je v `firmware/zapper_v5/zapper_v5.ino` a jeho popis v `firmware/zapper_v5/README.md`. Po nahrání firmwaru:

1. Otevřete zobrazení **Zařízení**.
2. Vyberte port COM a klikněte na **Připojit**.
3. Počkejte na stav **Připraveno**.
4. Odešlete dnešní relaci nebo spusťte jednu hodnotu v ručním režimu.
5. Zkontrolujte připojení na desce a poté stiskněte její fyzické tlačítko; teprve potom se výstup spustí.

Vybraný port se pamatuje v místním souboru `data/device.json`. Profilové relace ukládají samostatné a přesné `device_steps`; popis jako „30 kHz“ zůstává čitelným textem, zatímco deska obdrží `30000000` millihertzů a dobu v milisekundách.

### Jazyky LCD firmwaru

Firmware 5.1.0 má 30 samostatných jazykových variant vytvářených z jedné kódové základny. Každý Arduino sketch obsahuje pouze jednu sadu textů LCD. Jazyky používající latinku mají vlastní krátké texty uložené jako bezpečné ASCII. Pro azbuku a další písma, která typický LCD1602/HD44780 neumí spolehlivě zobrazit, používá odpovídající varianta anglické rozhraní LCD. Úplný seznam je v `firmware/LANGUAGES.md`.

Příkaz `go run ./tools/firmware_i18n` vytvoří všechny sketche v `build/generated/firmware/`. Běžný proces `build.ps1` to provede automaticky a zahrne varianty do portable verze.

### Nahrání firmwaru z aplikace

Sekce **Zařízení → Firmware** zobrazuje zjištěnou verzi, nejnovější verzi, jazyk varianty firmwaru a jazyk LCD. Uživatel vybere nový nebo starý bootloader Arduino Nano a výslovně klikne na **Nahrát firmware**; aplikace při spuštění nikdy automaticky firmware do desky nenahrává.

Kompilaci a nahrávání zajišťuje `arduino-cli`. Zapper jej hledá v `tools/arduino-cli/`, vedle EXE, v `PATH` a v obvyklých umístěních Arduino IDE. Pokud nástroj není dostupný, aplikace to jasně oznámí a tlačítko nahrávání zůstane neaktivní. Kompilace také vyžaduje core `arduino:avr` a knihovnu `LiquidCrystal_I2C` dostupné pro použitou instalaci `arduino-cli`.

### Zjištění jazyka a výběr firmwaru

Při spuštění aplikace načte jazyk prostředí WebView2/Windows (`navigator.languages`) a přiřadí jej k jednomu z 30 podporovaných kódů. Pokud systémový jazyk není podporován, vybere se angličtina. V režimu **Automaticky (Windows)** se jazyk kontroluje při každém spuštění; ruční volba se ukládá do `data/settings.json`, dokud není automatický režim znovu zapnut.

Stejný jazykový kód je výchozí volbou na obrazovce pro nahrání firmwaru. U jazyků, které LCD1602 nepodporuje, aplikace stále vybere variantu označenou jazykem uživatele, ale informuje, že text na LCD bude anglicky. Firmware se při startu aplikace nikdy nenahrává automaticky; nahrání vyžaduje výslovné kliknutí uživatele, aby se omylem nepřepsal jiný program, který již na Arduinu je.

## Sestavení

Je vyžadován Go. Nejjednodušší je v kořenové složce projektu spustit:

```text
build.bat
```

Případně v PowerShellu:

```powershell
.\build.ps1
```

Skript spustí testy a analýzu kódu, sestaví `build/generated/Zapper-dev.exe` a připraví portable `build/Zapper/Zapper.exe` bez konzolového okna.

## Struktura projektu

- `app/` — Go kód, rozhraní HTML/CSS/JS, návod a databáze frekvencí.
- `firmware/zapper_v5/` — aktuální firmware Arduino.
- `data/` — aktivní profily, průběh, archiv, nastavení zařízení a automatické zálohy.
- `locales/` — verzované překlady rozhraní a návodu používané při vývoji a kopírované do vydání.
- `build/Zapper/` — hotová portable verze ke zkopírování na jiný počítač.