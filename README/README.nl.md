**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

De nieuwe versie van de toepassing draait in één venster en vereist geen Python, Node.js of Wails. Ze kan zonder aangesloten bord als planner en logboek worden gebruikt, of een Arduino Nano via USB aansturen.

## Licentie en verantwoordelijkheid

De code, firmware, schema’s en documentatie zijn publiek beschikbaar voor niet-commercieel gebruik onder de licentie **PolyForm Noncommercial 1.0.0**. Ze mogen worden gebruikt, bestudeerd, aangepast en verspreid voor doeleinden die volgens die licentie zijn toegestaan, maar het project mag niet commercieel worden gebruikt zonder afzonderlijke toestemming van de auteur. Zie het bestand `LICENSE` voor details.

Het project wordt zonder garantie aangeboden voor zelfstandige experimenten en DIY-toepassingen. De gebruiker is verantwoordelijk voor correcte montage, wijzigingen en de manier waarop het apparaat wordt gebruikt. De auteur is niet aansprakelijk voor schade aan hardware, andere schade of gevolgen van onjuiste montage of onjuist gebruik en garandeert geen bepaalde gezondheidseffecten.

## De toepassing starten

Start `Zapper.exe` vanuit de map van de portable versie. Vaste personen en hun identificatoren worden opgeslagen in `data/persons.json`, actieve profielen in `data/profiles.json` en elke run heeft een eigen bestand in `data/progress/`. Voltooide runs worden verplaatst naar mappen `data/archive/<id>/` met `profile.json` en `progress.json`. Bordinstellingen staan in `data/device.json`, terwijl toepassingsinstellingen, waaronder de gedetecteerde of gekozen taal, in `data/settings.json` staan. Back-ups blijven in lokale submappen `backups/`. Alles staat naast het EXE-bestand; er wordt niets naar AppData, Documenten of het Windows-register geschreven.

In de weergave **Profielen** kun je personen toevoegen, kant-en-klare contexttekst voor AI naar het klembord genereren en vereenvoudigde JSON plakken die door een AI-model is teruggegeven. Frequenties worden in dit formaat opgegeven als `frequency_hz`; de toepassing valideert het profiel, toont een voorbeeld en maakt pas na bevestiging een nieuwe `run_id`. De vorige actieve run van die persoon wordt eerst gearchiveerd.

Tijdens een profielsessie slaat de knop **Pauze** het resterende deel van de huidige stap en alle volgende stappen op in de lokale voortgang. Hervatten stuurt een verkorte reeks naar de ongewijzigde firmware en vereist opnieuw fysieke bevestiging op het bord. **Stoppen** annuleert de gedeeltelijke voortgang en laat de volledige sessie beschikbaar om opnieuw uit te voeren.

Overgeslagen sessies blijven als achterstallig in de wachtrij staan. Programmaregels bepalen het aantal delen, pauzes binnen een reeks, de afstand tussen volledige sessies, de afkoelperiode na een sessie en de compatibiliteit met andere programma’s op dezelfde dag. Een profiel zonder achterstallige sessies wordt automatisch gearchiveerd zodra het plan is voltooid; met **Programma beëindigen** kan het eerder worden afgesloten.

## Taal van de toepassing

Bij het opstarten leest de toepassing de taal van Windows/WebView2 en koppelt die aan een van de 30 ondersteunde talen. Zolang **Automatisch (Windows)** is ingesteld, wordt de taal bij elke start opnieuw gedetecteerd. Een handmatige taalkeuze in het linkerpaneel wordt opgeslagen in `data/settings.json` en schakelt automatische wijzigingen uit totdat de automatische modus opnieuw wordt gekozen.

De taal van de toepassing is ook de standaardtaal van de firmwarevariant. Voor schriftsystemen die een standaard LCD1602/HD44780 niet betrouwbaar kan weergeven, kiest de toepassing de overeenkomstige firmwarevariant met Engelse LCD-tekst; de desktopinterface blijft de gekozen taal gebruiken.

## Arduino en USB

De huidige firmware staat in `firmware/zapper_v5/zapper_v5.ino`, met een beschrijving in `firmware/zapper_v5/README.md`. Nadat de firmware is geflasht:

1. Open de weergave **Apparaat**.
2. Selecteer de COM-poort en klik op **Verbinden**.
3. Wacht op de status **Gereed**.
4. Stuur de sessie van vandaag of start één waarde in handmatige modus.
5. Controleer de aansluitingen op het bord en druk daarna op de fysieke knop; pas dan begint de uitgang.

De gekozen poort wordt onthouden in het lokale bestand `data/device.json`. Profielsessies bewaren afzonderlijke, exacte `device_steps`; een beschrijving zoals “30 kHz” blijft leesbare tekst, terwijl het bord `30000000` millihertz en de duur in milliseconden ontvangt.

### Talen van de LCD-firmware

Firmware 5.1.0 heeft 30 afzonderlijke taalvarianten die uit één codebasis worden gegenereerd. Elke Arduino-sketch bevat slechts één set LCD-teksten. Talen met het Latijnse alfabet hebben hun eigen korte teksten die veilig als ASCII worden opgeslagen. Voor Cyrillisch en andere schriftsystemen die een typische LCD1602/HD44780 niet betrouwbaar kan weergeven, gebruikt de overeenkomstige variant een Engelse LCD-interface. De volledige lijst staat in `firmware/LANGUAGES.md`.

De opdracht `go run ./tools/firmware_i18n` maakt alle sketches in `build/generated/firmware/`. Het normale `build.ps1`-proces doet dit automatisch en neemt de varianten op in de portable versie.

### Firmware flashen vanuit de toepassing

Het gedeelte **Apparaat → Firmware** toont de gedetecteerde versie, de nieuwste versie, de taal van de firmwarevariant en de LCD-taal. De gebruiker kiest de nieuwe of oude Arduino Nano-bootloader en klikt expliciet op **Firmware flashen**; de toepassing flasht het bord nooit automatisch tijdens het opstarten.

Compilatie en upload worden uitgevoerd door `arduino-cli`. Zapper zoekt ernaar in `tools/arduino-cli/`, naast het EXE-bestand, in `PATH` en in gebruikelijke Arduino IDE-locaties. Als het hulpprogramma niet beschikbaar is, meldt de toepassing dit duidelijk en blijft de flashknop uitgeschakeld. Voor compilatie moeten ook de core `arduino:avr` en de bibliotheek `LiquidCrystal_I2C` beschikbaar zijn voor de gebruikte `arduino-cli`-installatie.

### Taaldetectie en firmwarekeuze

Bij het opstarten leest de toepassing de taal van de WebView2/Windows-omgeving (`navigator.languages`) en koppelt die aan een van de 30 ondersteunde codes. Als de systeemtaal niet wordt ondersteund, wordt Engels gekozen. In de modus **Automatisch (Windows)** wordt de taal bij elke start gecontroleerd; een handmatige keuze wordt in `data/settings.json` opgeslagen totdat de automatische modus opnieuw wordt ingeschakeld.

Dezelfde taalcode is de standaardkeuze op het firmware-flashscherm. Voor talen die LCD1602 niet ondersteunt, kiest de toepassing nog steeds de variant die bij de taal van de gebruiker hoort, maar meldt ze dat de LCD-tekst Engels zal zijn. Firmware wordt nooit automatisch geflasht wanneer de toepassing start; flashen vereist een expliciete klik van de gebruiker zodat een ander programma dat al op de Arduino staat niet per ongeluk wordt overschreven.

## Bouwen

Go is vereist. De eenvoudigste manier is om in de hoofdmap van het project uit te voeren:

```text
build.bat
```

Of in PowerShell:

```powershell
.\build.ps1
```

Het script voert tests en codeanalyse uit, bouwt `build/generated/Zapper-dev.exe` en bereidt de portable `build/Zapper/Zapper.exe` zonder consolevenster voor.

## Projectindeling

- `app/` — Go-code, HTML/CSS/JS-interface, handleiding en frequentiedatabase.
- `firmware/zapper_v5/` — huidige Arduino-firmware.
- `data/` — actieve profielen, voortgang, archief, apparaatinstellingen en automatische back-ups.
- `locales/` — versiebeheerde vertalingen van de interface en handleiding, gebruikt bij ontwikkeling en gekopieerd naar releases.
- `build/Zapper/` — kant-en-klare portable versie om naar een andere computer te kopiëren.