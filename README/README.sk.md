**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Nová verzia aplikácie funguje v jednom okne a nevyžaduje Python, Node.js ani Wails. Možno ju používať ako plánovač a denník bez pripojenej dosky alebo na ovládanie Arduino Nano cez USB.

## Licencia a zodpovednosť

Kód, firmware, schémy a dokumentácia sú verejne dostupné na nekomerčné použitie pod licenciou **PolyForm Noncommercial 1.0.0**. Možno ich používať, skúmať, upravovať a šíriť na účely povolené touto licenciou, projekt však nemožno komerčne používať bez osobitného súhlasu autora. Podrobnosti nájdete v súbore `LICENSE`.

Projekt sa poskytuje bez záruky na vlastné experimenty a DIY použitie. Používateľ zodpovedá za správne zostavenie, úpravy a spôsob používania zariadenia. Autor nezodpovedá za poškodenie hardvéru, iné škody ani následky nesprávneho zostavenia alebo používania a nezaručuje žiadne konkrétne zdravotné účinky.

## Spustenie aplikácie

Spustite `Zapper.exe` z priečinka portable verzie. Trvalé osoby a ich identifikátory sú uložené v `data/persons.json`, aktívne profily v `data/profiles.json` a každý beh má vlastný súbor v `data/progress/`. Dokončené behy sa presúvajú do priečinkov `data/archive/<id>/`, ktoré obsahujú `profile.json` a `progress.json`. Nastavenia dosky sa ukladajú v `data/device.json`, zatiaľ čo nastavenia aplikácie vrátane zisteného alebo vybraného jazyka sú v `data/settings.json`. Zálohy zostávajú v miestnych podpriečinkoch `backups/`. Všetko je vedľa súboru EXE; nič sa nezapisuje do AppData, Dokumentov ani registra Windows.

V zobrazení **Profily** môžete pridávať osoby, vytvárať pripravený kontextový text pre AI do schránky a vkladať zjednodušený JSON vrátený modelom AI. Frekvencie sa v tomto formáte zadávajú ako `frequency_hz`; aplikácia profil overí, zobrazí náhľad a nový `run_id` vytvorí až po potvrdení. Predchádzajúci aktívny beh danej osoby sa najprv archivuje.

Počas profilovej relácie tlačidlo **Pauza** uloží zostávajúcu časť aktuálneho kroku a všetky nasledujúce kroky do miestneho priebehu. Pri pokračovaní sa do nezmeneného firmwaru odošle skrátená sekvencia a znova je potrebné fyzické potvrdenie na doske. **Zastaviť** zruší čiastočný priebeh a ponechá celú reláciu pripravenú na opätovné spustenie.

Vynechané relácie zostávajú vo fronte ako omeškané. Pravidlá programu určujú počet častí, prestávky v rámci série, odstup medzi úplnými reláciami, čas zotavenia po relácii a kompatibilitu s inými programami v ten istý deň. Profil bez omeškaných relácií sa po dokončení plánu automaticky archivuje, zatiaľ čo **Ukončiť program** umožňuje uzavrieť ho skôr.

## Jazyk aplikácie

Pri spustení aplikácia načíta jazyk Windows/WebView2 a priradí ho k jednému z 30 podporovaných jazykov. Kým je nastavený režim **Automaticky (Windows)**, detekcia jazyka sa vykonáva pri každom spustení. Ručný výber jazyka v ľavom paneli sa uloží do `data/settings.json` a vypne automatické zmeny, kým sa znova nezvolí automatický režim.

Jazyk aplikácie je zároveň predvoleným jazykom variantu firmwaru. Pre písma, ktoré štandardný LCD1602/HD44780 nedokáže spoľahlivo zobrazovať, aplikácia vyberie zodpovedajúci variant firmwaru s anglickými textami LCD; desktopové rozhranie naďalej používa zvolený jazyk.

## Arduino a USB

Aktuálny firmware sa nachádza v `firmware/zapper_v5/zapper_v5.ino` a jeho popis v `firmware/zapper_v5/README.md`. Po nahratí firmwaru:

1. Otvorte zobrazenie **Zariadenie**.
2. Vyberte port COM a kliknite na **Pripojiť**.
3. Počkajte na stav **Pripravené**.
4. Odošlite dnešnú reláciu alebo spustite jednu hodnotu v manuálnom režime.
5. Skontrolujte pripojenia na doske a potom stlačte jej fyzické tlačidlo; až potom sa spustí výstup.

Vybraný port sa zapamätá v miestnom súbore `data/device.json`. Profilové relácie ukladajú samostatné a presné `device_steps`; popis ako „30 kHz“ zostáva čitateľným textom, zatiaľ čo doska dostane `30000000` millihertzov a čas v milisekundách.

### Jazyky LCD firmwaru

Firmware 5.1.0 má 30 samostatných jazykových variantov vytvorených z jednej kódovej základne. Každý Arduino sketch obsahuje iba jednu sadu textov LCD. Jazyky používajúce latinku majú vlastné krátke texty uložené ako bezpečné ASCII. Pre cyriliku a iné písma, ktoré typický LCD1602/HD44780 nedokáže spoľahlivo zobrazovať, používa príslušný variant anglické LCD rozhranie. Úplný zoznam je v `firmware/LANGUAGES.md`.

Príkaz `go run ./tools/firmware_i18n` vytvorí všetky sketche v `build/generated/firmware/`. Bežný proces `build.ps1` to vykoná automaticky a zahrnie varianty do portable verzie.

### Nahrávanie firmwaru z aplikácie

Časť **Zariadenie → Firmware** zobrazuje zistenú verziu, najnovšiu verziu, jazyk variantu firmwaru a jazyk LCD. Používateľ vyberie nový alebo starý bootloader Arduino Nano a výslovne klikne na **Nahrať firmware**; aplikácia pri spustení nikdy automaticky firmware do dosky nenahráva.

Kompiláciu a nahrávanie zabezpečuje `arduino-cli`. Zapper ho hľadá v `tools/arduino-cli/`, vedľa EXE, v `PATH` a v bežných umiestneniach Arduino IDE. Ak nástroj nie je dostupný, aplikácia to jasne oznámi a tlačidlo nahrávania zostane neaktívne. Kompilácia tiež vyžaduje core `arduino:avr` a knižnicu `LiquidCrystal_I2C` dostupné pre použitú inštaláciu `arduino-cli`.

### Zistenie jazyka a výber firmwaru

Pri spustení aplikácia načíta jazyk prostredia WebView2/Windows (`navigator.languages`) a priradí ho k jednému z 30 podporovaných kódov. Ak systémový jazyk nie je podporovaný, vyberie sa angličtina. V režime **Automaticky (Windows)** sa jazyk kontroluje pri každom spustení; ručný výber sa uloží do `data/settings.json`, kým sa automatický režim znova nezapne.

Rovnaký kód jazyka je predvolenou voľbou na obrazovke nahrávania firmwaru. Pri jazykoch, ktoré LCD1602 nepodporuje, aplikácia stále vyberie variant označený jazykom používateľa, ale informuje, že text LCD bude v angličtine. Firmware sa pri štarte aplikácie nikdy nenahráva automaticky; nahrávanie vyžaduje výslovné kliknutie používateľa, aby sa omylom neprepísal iný program už uložený na Arduine.

## Zostavenie

Vyžaduje sa Go. Najjednoduchšie je v koreňovom priečinku projektu spustiť:

```text
build.bat
```

Prípadne v PowerShelli:

```powershell
.\build.ps1
```

Skript spustí testy a analýzu kódu, zostaví `build/generated/Zapper-dev.exe` a pripraví portable `build/Zapper/Zapper.exe` bez konzolového okna.

## Štruktúra projektu

- `app/` — Go kód, rozhranie HTML/CSS/JS, návod a databáza frekvencií.
- `firmware/zapper_v5/` — aktuálny firmware Arduino.
- `data/` — aktívne profily, priebeh, archív, nastavenia zariadenia a automatické zálohy.
- `locales/` — verzované preklady rozhrania a návodu používané pri vývoji a kopírované do vydaní.
- `build/Zapper/` — hotová portable verzia na skopírovanie do iného počítača.