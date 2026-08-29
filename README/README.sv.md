**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Den nya versionen av programmet körs i ett enda fönster och kräver varken Python, Node.js eller Wails. Den kan användas som planeringsverktyg och loggbok utan anslutet kort, eller styra en Arduino Nano via USB.

## Licens och ansvar

Kod, firmware, kopplingsscheman och dokumentation är offentligt tillgängliga för icke-kommersiell användning enligt licensen **PolyForm Noncommercial 1.0.0**. De får användas, studeras, ändras och spridas för de ändamål som licensen tillåter, men projektet får inte användas kommersiellt utan separat tillstånd från upphovspersonen. Se filen `LICENSE` för mer information.

Projektet tillhandahålls utan garanti för egna experiment och DIY-användning. Användaren ansvarar för korrekt montering, ändringar och hur enheten används. Upphovspersonen ansvarar inte för hårdvaruskador, andra skador eller följder av felaktig montering eller användning och garanterar inga särskilda hälsoeffekter.

## Starta programmet

Kör `Zapper.exe` från mappen för den portabla versionen. Fasta personer och deras identifierare sparas i `data/persons.json`, aktiva profiler i `data/profiles.json` och varje körning har en egen fil i `data/progress/`. Avslutade körningar flyttas till mapparna `data/archive/<id>/`, som innehåller `profile.json` och `progress.json`. Kortets inställningar sparas i `data/device.json`, medan programmets inställningar, inklusive upptäckt eller vald språkversion, finns i `data/settings.json`. Säkerhetskopior ligger kvar i lokala undermappar `backups/`. Allt ligger bredvid EXE-filen; inget skrivs till AppData, Dokument eller Windows-registret.

I vyn **Profiler** kan du lägga till personer, skapa färdig kontexttext för AI att kopiera och klistra in förenklad JSON som returnerats av en AI-modell. Frekvenser anges i detta format som `frequency_hz`; programmet validerar profilen, visar en förhandsgranskning och skapar en ny `run_id` först efter bekräftelse. Personens tidigare aktiva körning arkiveras först.

Under en profilsession sparar knappen **Paus** den återstående delen av aktuellt steg och alla följande steg i den lokala progressen. Vid återupptagning skickas en förkortad sekvens till oförändrad firmware och fysisk bekräftelse på kortet krävs igen. **Stoppa** avbryter den delvisa progressen och lämnar hela sessionen tillgänglig för att köras på nytt.

Överhoppade sessioner ligger kvar i kön som försenade. Programreglerna anger antal delar, pauser inom en serie, avstånd mellan fullständiga sessioner, återhämtningstid efter en session samt kompatibilitet med andra program samma dag. En profil utan försenade sessioner arkiveras automatiskt när planen är slutförd, medan **Avsluta program** gör det möjligt att stänga den tidigare.

## Programspråk

Vid start läser programmet språket i Windows/WebView2 och kopplar det till ett av 30 språk som stöds. Så länge inställningen är **Automatiskt (Windows)** görs språkidentifieringen vid varje start. Ett manuellt språkval i vänsterpanelen sparas i `data/settings.json` och stänger av automatiska ändringar tills automatiskt läge väljs igen.

Programs språket är också standardspråk för firmwarevarianten. För skriftsystem som en vanlig LCD1602/HD44780 inte kan visa på ett portabelt sätt väljer programmet motsvarande firmwarevariant med engelska LCD-texter; skrivbordsgränssnittet fortsätter använda det valda språket.

## Arduino och USB

Aktuell firmware finns i `firmware/zapper_v5/zapper_v5.ino` och beskrivningen i `firmware/zapper_v5/README.md`. Efter att firmware har flashats:

1. Öppna vyn **Enhet**.
2. Välj COM-port och klicka på **Anslut**.
3. Vänta tills statusen är **Redo**.
4. Skicka dagens session eller starta ett enskilt värde i manuellt läge.
5. Kontrollera anslutningarna på kortet och tryck sedan på den fysiska knappen; först då startar utgången.

Den valda porten sparas i den lokala filen `data/device.json`. Profilsessioner lagrar separata, exakta `device_steps`; en beskrivning som ”30 kHz” är fortfarande läsbar text, medan kortet får `30000000` millihertz och tiden i millisekunder.

### Språk för LCD-firmware

Firmware 5.1.0 har 30 separata språkvarianter som skapas från samma kodbas. Varje Arduino-sketch innehåller bara en uppsättning LCD-texter. Språk som använder latinskt alfabet har egna korta texter lagrade som säker ASCII. För kyrilliska och andra skriftsystem som en typisk LCD1602/HD44780 inte kan visa portabelt använder motsvarande variant ett engelskt LCD-gränssnitt. Hela listan finns i `firmware/LANGUAGES.md`.

Kommandot `go run ./tools/firmware_i18n` skapar alla sketcher i `build/generated/firmware/`. Det vanliga `build.ps1`-flödet gör detta automatiskt och inkluderar varianterna i den portabla versionen.

### Flasha firmware från programmet

Avsnittet **Enhet → Firmware** visar identifierad version, senaste version, firmwarevariantens språk och LCD-språk. Användaren väljer den nya eller gamla bootloadern för Arduino Nano och klickar uttryckligen på **Flasha firmware**; programmet flashar aldrig kortet automatiskt vid start.

Kompilering och uppladdning hanteras av `arduino-cli`. Zapper söker efter det i `tools/arduino-cli/`, bredvid EXE-filen, i `PATH` och på vanliga platser för Arduino IDE. Om verktyget saknas visar programmet detta tydligt och flashknappen förblir inaktiverad. Kompilering kräver också att kärnan `arduino:avr` och biblioteket `LiquidCrystal_I2C` är tillgängliga för den använda `arduino-cli`-installationen.

### Språkidentifiering och val av firmware

Vid start läser programmet språket i WebView2/Windows-miljön (`navigator.languages`) och kopplar det till en av de 30 stödda språkkoderna. Om systemspråket inte stöds väljs engelska. I läget **Automatiskt (Windows)** kontrolleras språket vid varje start; ett manuellt val sparas i `data/settings.json` tills automatiskt läge aktiveras igen.

Samma språkkod är standardval på skärmen för firmwareflashning. För språk som LCD1602 inte stöder väljer programmet ändå varianten som motsvarar användarens språk, men informerar om att LCD-texten blir engelsk. Firmware flashas aldrig automatiskt när programmet startar; flashning kräver ett uttryckligt klick från användaren för att inte av misstag skriva över ett annat program som redan finns på Arduino.

## Bygga

Go krävs. Enklast är att köra följande i projektets rotmapp:

```text
build.bat
```

Alternativt i PowerShell:

```powershell
.\build.ps1
```

Skriptet kör tester och kodanalys, bygger `build/generated/Zapper-dev.exe` och förbereder den portabla `build/Zapper/Zapper.exe` utan konsolfönster.

## Projektstruktur

- `app/` — Go-kod, HTML/CSS/JS-gränssnitt, guide och frekvensdatabas.
- `firmware/zapper_v5/` — aktuell Arduino-firmware.
- `data/` — aktiva profiler, progress, arkiv, enhetsinställningar och automatiska säkerhetskopior.
- `locales/` — versionshanterade översättningar för gränssnittet och guiden, använda vid utveckling och kopierade till releaser.
- `build/Zapper/` — färdig portabel version att kopiera till en annan dator.