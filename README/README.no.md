**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Den nye versjonen av programmet kjører i ett vindu og krever verken Python, Node.js eller Wails. Den kan brukes som planlegger og loggbok uten et tilkoblet kort, eller styre en Arduino Nano via USB.

## Lisens og ansvar

Kode, firmware, koblingsskjemaer og dokumentasjon er offentlig tilgjengelige for ikke-kommersiell bruk under lisensen **PolyForm Noncommercial 1.0.0**. De kan brukes, studeres, endres og distribueres til formål som lisensen tillater, men prosjektet kan ikke brukes kommersielt uten særskilt tillatelse fra forfatteren. Se filen `LICENSE` for detaljer.

Prosjektet leveres uten garanti for egne eksperimenter og DIY-bruk. Brukeren er ansvarlig for korrekt montering, endringer og måten enheten brukes på. Forfatteren er ikke ansvarlig for skader på maskinvare, andre tap eller følger av feil montering eller bruk, og garanterer ingen bestemte helseeffekter.

## Starte programmet

Kjør `Zapper.exe` fra mappen til den portable versjonen. Faste personer og identifikatorene deres lagres i `data/persons.json`, aktive profiler i `data/profiles.json`, og hver kjøring har sin egen fil i `data/progress/`. Fullførte kjøringer flyttes til mapper `data/archive/<id>/` som inneholder `profile.json` og `progress.json`. Kortinnstillinger lagres i `data/device.json`, mens programinnstillinger, inkludert oppdaget eller valgt språk, lagres i `data/settings.json`. Sikkerhetskopier blir liggende i lokale undermapper `backups/`. Alt ligger ved siden av EXE-filen; ingenting skrives til AppData, Dokumenter eller Windows-registeret.

I visningen **Profiler** kan du legge til personer, generere ferdig konteksttekst for AI til utklippstavlen og lime inn forenklet JSON returnert av en AI-modell. Frekvenser oppgis i dette formatet som `frequency_hz`; programmet validerer profilen, viser en forhåndsvisning og oppretter en ny `run_id` først etter bekreftelse. Personens forrige aktive kjøring arkiveres først.

Under en profiløkt lagrer knappen **Pause** den gjenværende delen av gjeldende trinn og alle påfølgende trinn i lokal fremdrift. Når økten fortsettes, sendes en forkortet sekvens til uendret firmware, og fysisk bekreftelse på kortet kreves på nytt. **Stopp** avbryter delvis fremdrift og lar hele økten være tilgjengelig for å kjøres igjen.

Hoppede økter blir liggende i køen som forsinkede. Programreglene bestemmer antall deler, pauser innenfor en serie, avstand mellom komplette økter, nedkjølingstid etter en økt og kompatibilitet med andre programmer samme dag. En profil uten forsinkede økter arkiveres automatisk når planen er fullført, mens **Avslutt program** gjør det mulig å lukke den tidligere.

## Programspråk

Ved oppstart leser programmet språket i Windows/WebView2 og kobler det til ett av 30 støttede språk. Så lenge innstillingen står på **Automatisk (Windows)**, utføres språkdeteksjonen ved hver oppstart. Et manuelt språkvalg i venstrepanelet lagres i `data/settings.json` og deaktiverer automatiske endringer til automatisk modus velges igjen.

Programspråket er også standardspråket for firmwarevarianten. For skriftsystemer som en standard LCD1602/HD44780 ikke kan vise på en portabel måte, velger programmet den tilsvarende firmwarevarianten med engelsk LCD-tekst; skrivebordsgrensesnittet fortsetter å bruke det valgte språket.

## Arduino og USB

Gjeldende firmware ligger i `firmware/zapper_v5/zapper_v5.ino`, med beskrivelse i `firmware/zapper_v5/README.md`. Etter at firmwaren er lastet opp:

1. Åpne visningen **Enhet**.
2. Velg COM-port og klikk **Koble til**.
3. Vent på statusen **Klar**.
4. Send dagens økt eller start én enkelt verdi i manuell modus.
5. Kontroller tilkoblingene på kortet og trykk deretter på den fysiske knappen; først da starter utgangen.

Den valgte porten huskes i den lokale filen `data/device.json`. Profiløkter lagrer separate, nøyaktige `device_steps`; en beskrivelse som «30 kHz» forblir menneskelesbar tekst, mens kortet mottar `30000000` millihertz og varigheten i millisekunder.

### Språk for LCD-firmware

Firmware 5.1.0 har 30 separate språkvarianter som genereres fra én kodebase. Hver Arduino-sketch inneholder bare ett sett med LCD-tekster. Språk som bruker det latinske alfabetet har egne korte tekster lagret som sikker ASCII. For kyrillisk og andre skriftsystemer som en typisk LCD1602/HD44780 ikke kan vise portabelt, bruker den tilsvarende varianten et engelsk LCD-grensesnitt. Hele listen finnes i `firmware/LANGUAGES.md`.

Kommandoen `go run ./tools/firmware_i18n` lager alle sketchene i `build/generated/firmware/`. Den vanlige `build.ps1`-prosessen gjør dette automatisk og inkluderer variantene i den portable versjonen.

### Laste opp firmware fra programmet

Delen **Enhet → Firmware** viser oppdaget versjon, nyeste versjon, språket til firmwarevarianten og LCD-språket. Brukeren velger ny eller gammel bootloader for Arduino Nano og klikker uttrykkelig på **Last opp firmware**; programmet skriver aldri firmware automatisk til kortet ved oppstart.

Kompilering og opplasting håndteres av `arduino-cli`. Zapper leter etter verktøyet i `tools/arduino-cli/`, ved siden av EXE-filen, i `PATH` og på vanlige Arduino IDE-plasseringer. Hvis verktøyet ikke er tilgjengelig, viser programmet dette tydelig og opplastingsknappen forblir deaktivert. Kompilering krever også at kjernen `arduino:avr` og biblioteket `LiquidCrystal_I2C` er tilgjengelige for `arduino-cli`-installasjonen som brukes.

### Språkdeteksjon og valg av firmware

Ved oppstart leser programmet språket i WebView2/Windows-miljøet (`navigator.languages`) og kobler det til en av de 30 støttede kodene. Hvis systemspråket ikke støttes, velges engelsk. I modusen **Automatisk (Windows)** kontrolleres språket ved hver oppstart; et manuelt valg lagres i `data/settings.json` til automatisk modus aktiveres igjen.

Den samme språkkoden er standardvalget på skjermen for firmwareopplasting. For språk som LCD1602 ikke støtter, velger programmet fortsatt varianten som er merket med brukerens språk, men informerer om at LCD-teksten vil være engelsk. Firmware lastes aldri opp automatisk når programmet starter; opplasting krever et uttrykkelig klikk fra brukeren, slik at et annet program som allerede ligger på Arduino ikke overskrives ved et uhell.

## Bygging

Go er nødvendig. Det enkleste er å kjøre følgende i prosjektets rotmappe:

```text
build.bat
```

Alternativt i PowerShell:

```powershell
.\build.ps1
```

Skriptet kjører tester og kodeanalyse, bygger `build/generated/Zapper-dev.exe` og klargjør den portable `build/Zapper/Zapper.exe` uten konsollvindu.

## Prosjektstruktur

- `app/` — Go-kode, HTML/CSS/JS-grensesnitt, veiledning og frekvensdatabase.
- `firmware/zapper_v5/` — gjeldende Arduino-firmware.
- `data/` — aktive profiler, fremdrift, arkiv, enhetsinnstillinger og automatiske sikkerhetskopier.
- `locales/` — versjonsstyrte oversettelser av grensesnittet og veiledningen, brukt under utvikling og kopiert til releaser.
- `build/Zapper/` — ferdig portabel versjon som kan kopieres til en annen datamaskin.