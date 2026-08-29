**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Den nye version af programmet kører i ét vindue og kræver hverken Python, Node.js eller Wails. Det kan bruges som planlægger og logbog uden et tilsluttet kort eller styre en Arduino Nano via USB.

## Licens og ansvar

Kode, firmware, diagrammer og dokumentation er offentligt tilgængelige til ikke-kommerciel brug under licensen **PolyForm Noncommercial 1.0.0**. De må bruges, undersøges, ændres og distribueres til formål, som licensen tillader, men projektet må ikke bruges kommercielt uden særskilt tilladelse fra forfatteren. Se filen `LICENSE` for detaljer.

Projektet leveres uden garanti til selvstændige eksperimenter og DIY-brug. Brugeren er ansvarlig for korrekt montering, ændringer og den måde, enheden bruges på. Forfatteren er ikke ansvarlig for skader på hardware, andre tab eller følger af forkert montering eller brug og garanterer ingen bestemte helbredseffekter.

## Start programmet

Kør `Zapper.exe` fra mappen med den portable version. Faste personer og deres identifikatorer gemmes i `data/persons.json`, aktive profiler i `data/profiles.json`, og hver kørsel har sin egen fil i `data/progress/`. Afsluttede kørsler flyttes til mapper `data/archive/<id>/`, som indeholder `profile.json` og `progress.json`. Kortets indstillinger gemmes i `data/device.json`, mens programindstillinger, herunder det registrerede eller valgte sprog, gemmes i `data/settings.json`. Sikkerhedskopier bliver liggende i lokale undermapper `backups/`. Alt ligger ved siden af EXE-filen; intet skrives til AppData, Dokumenter eller Windows-registreringsdatabasen.

I visningen **Profiler** kan du tilføje personer, generere færdig AI-konteksttekst til udklipsholderen og indsætte forenklet JSON returneret af en AI-model. Frekvenser angives i dette format som `frequency_hz`; programmet validerer profilen, viser en forhåndsvisning og opretter først en ny `run_id` efter bekræftelse. Personens tidligere aktive kørsel arkiveres først.

Under en profilsession gemmer knappen **Pause** den resterende del af det aktuelle trin og alle efterfølgende trin i den lokale fremdrift. Ved genoptagelse sendes en forkortet sekvens til den uændrede firmware, og fysisk bekræftelse på kortet kræves igen. **Stop** annullerer delvis fremdrift og efterlader hele sessionen klar til at blive kørt igen.

Springede sessioner bliver i køen som forsinkede. Programreglerne fastlægger antal dele, pauser inden for en serie, afstand mellem fulde sessioner, nedkøling efter en session og kompatibilitet med andre programmer samme dag. En profil uden forsinkede sessioner arkiveres automatisk, når planen er gennemført, mens **Afslut program** gør det muligt at lukke den tidligere.

## Programsprog

Ved opstart læser programmet sproget i Windows/WebView2 og matcher det med et af 30 understøttede sprog. Så længe indstillingen står på **Automatisk (Windows)**, udføres sprogregistrering ved hver start. Et manuelt sprogvalg i venstre panel gemmes i `data/settings.json` og deaktiverer automatiske ændringer, indtil automatisk tilstand vælges igen.

Programsproget er også standardsproget for firmwarevarianten. For skriftsystemer, som en almindelig LCD1602/HD44780 ikke kan vise på en portabel måde, vælger programmet den tilsvarende firmwarevariant med engelsk LCD-tekst; desktopgrænsefladen fortsætter med at bruge det valgte sprog.

## Arduino og USB

Den aktuelle firmware findes i `firmware/zapper_v5/zapper_v5.ino`, og beskrivelsen findes i `firmware/zapper_v5/README.md`. Efter firmware er indlæst:

1. Åbn visningen **Enhed**.
2. Vælg COM-porten og klik på **Forbind**.
3. Vent på tilstanden **Klar**.
4. Send dagens session eller start en enkelt værdi i manuel tilstand.
5. Kontroller tilslutningerne på kortet og tryk derefter på den fysiske knap; først da starter udgangen.

Den valgte port huskes i den lokale fil `data/device.json`. Profilsessioner gemmer separate, nøjagtige `device_steps`; en beskrivelse som “30 kHz” forbliver menneskelæsbar tekst, mens kortet modtager `30000000` millihertz og varigheden i millisekunder.

### Sprog for LCD-firmware

Firmware 5.1.0 har 30 separate sprogvarianter, der genereres fra én kodebase. Hver Arduino-sketch indeholder kun ét sæt LCD-tekster. Sprog med latinsk alfabet har deres egne korte tekster gemt som sikker ASCII. For kyrillisk og andre skriftsystemer, som en typisk LCD1602/HD44780 ikke kan vise portabelt, bruger den tilsvarende variant en engelsk LCD-grænseflade. Hele listen findes i `firmware/LANGUAGES.md`.

Kommandoen `go run ./tools/firmware_i18n` opretter alle sketches i `build/generated/firmware/`. Den normale `build.ps1`-proces gør dette automatisk og inkluderer varianterne i den portable version.

### Indlæs firmware fra programmet

Afsnittet **Enhed → Firmware** viser den registrerede version, den nyeste version, firmwarevariantens sprog og LCD-sproget. Brugeren vælger den nye eller gamle bootloader til Arduino Nano og klikker udtrykkeligt på **Indlæs firmware**; programmet skriver aldrig automatisk firmware til kortet ved opstart.

Kompilering og upload håndteres af `arduino-cli`. Zapper leder efter værktøjet i `tools/arduino-cli/`, ved siden af EXE-filen, i `PATH` og på almindelige Arduino IDE-placeringer. Hvis værktøjet ikke er tilgængeligt, viser programmet det tydeligt, og uploadknappen forbliver deaktiveret. Kompilering kræver også, at kernen `arduino:avr` og biblioteket `LiquidCrystal_I2C` er tilgængelige for den anvendte `arduino-cli`-installation.

### Sprogregistrering og valg af firmware

Ved opstart læser programmet sproget i WebView2/Windows-miljøet (`navigator.languages`) og matcher det med en af de 30 understøttede koder. Hvis systemsproget ikke understøttes, vælges engelsk. I tilstanden **Automatisk (Windows)** kontrolleres sproget ved hver start; et manuelt valg gemmes i `data/settings.json`, indtil automatisk tilstand aktiveres igen.

Den samme sprogkode er standardvalget på skærmen til firmwareindlæsning. For sprog, som LCD1602 ikke understøtter, vælger programmet stadig varianten, der svarer til brugerens sprog, men oplyser, at LCD-teksten vil være engelsk. Firmware indlæses aldrig automatisk, når programmet starter; det kræver et udtrykkeligt klik fra brugeren, så et andet program, der allerede ligger på Arduino, ikke overskrives ved et uheld.

## Bygning

Go er påkrævet. Den nemmeste måde er at køre følgende i projektets rodmappe:

```text
build.bat
```

Alternativt i PowerShell:

```powershell
.\build.ps1
```

Scriptet kører tests og kodeanalyse, bygger `build/generated/Zapper-dev.exe` og klargør den portable `build/Zapper/Zapper.exe` uden konsolvindue.

## Projektstruktur

- `app/` — Go-kode, HTML/CSS/JS-grænseflade, vejledning og frekvensdatabase.
- `firmware/zapper_v5/` — aktuel Arduino-firmware.
- `data/` — aktive profiler, fremdrift, arkiv, enhedsindstillinger og automatiske sikkerhedskopier.
- `locales/` — versionsstyrede oversættelser af brugerfladen og vejledningen, brugt under udvikling og kopieret til releases.
- `build/Zapper/` — færdig portabel version, der kan kopieres til en anden computer.