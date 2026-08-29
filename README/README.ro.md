**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Noua versiune a aplicației rulează într-o singură fereastră și nu necesită Python, Node.js sau Wails. Poate fi folosită ca planificator și jurnal fără o placă conectată sau pentru a controla un Arduino Nano prin USB.

## Licență și responsabilitate

Codul, firmware-ul, schemele și documentația sunt disponibile public pentru utilizare necomercială sub licența **PolyForm Noncommercial 1.0.0**. Pot fi folosite, studiate, modificate și distribuite în scopurile permise de această licență, însă proiectul nu poate fi utilizat comercial fără acordul separat al autorului. Detaliile se află în fișierul `LICENSE`.

Proiectul este oferit fără garanție pentru experimente independente și utilizare DIY. Utilizatorul este responsabil pentru asamblarea corectă, modificări și modul de utilizare a dispozitivului. Autorul nu răspunde pentru deteriorarea hardware-ului, alte pierderi sau consecințe ale unei asamblări ori utilizări incorecte și nu garantează efecte specifice asupra sănătății.

## Pornirea aplicației

Porniți `Zapper.exe` din folderul versiunii portable. Persoanele persistente și identificatorii lor sunt stocate în `data/persons.json`, profilurile active în `data/profiles.json`, iar fiecare rulare are propriul fișier în `data/progress/`. Rulările finalizate sunt mutate în directoare `data/archive/<id>/`, care conțin `profile.json` și `progress.json`. Setările plăcii sunt în `data/device.json`, iar setările aplicației, inclusiv limba detectată sau selectată, în `data/settings.json`. Copiile de siguranță rămân în subdirectoarele locale `backups/`. Totul se află lângă fișierul EXE; nu se scrie nimic în AppData, Documente sau Registrul Windows.

În vizualizarea **Profiluri** puteți adăuga persoane, genera text de context pentru AI gata de copiat și lipi JSON simplificat returnat de un model AI. Frecvențele se furnizează în acest format ca `frequency_hz`; aplicația validează profilul, afișează o previzualizare și creează un nou `run_id` numai după confirmare. Rularea activă anterioară a persoanei este arhivată mai întâi.

În timpul unei sesiuni de profil, butonul **Pauză** salvează partea rămasă a pasului curent și toți pașii următori în progresul local. Reluarea trimite o secvență scurtată către firmware-ul neschimbat și necesită din nou confirmare fizică pe placă. **Oprire** anulează progresul parțial și lasă întreaga sesiune disponibilă pentru a fi executată din nou.

Sesiunile omise rămân în coadă ca restante. Regulile programului stabilesc numărul de părți, pauzele din interiorul unei serii, distanța dintre sesiunile complete, perioada de repaus după o sesiune și compatibilitatea cu alte programe din aceeași zi. Un profil fără sesiuni restante este arhivat automat după finalizarea planului, iar **Finalizare program** permite închiderea lui mai devreme.

## Limba aplicației

La pornire, aplicația citește limba Windows/WebView2 și o asociază cu una dintre cele 30 de limbi acceptate. Cât timp este selectat modul **Automat (Windows)**, detectarea limbii se face la fiecare pornire. Alegerea manuală a limbii din panoul din stânga este memorată în `data/settings.json` și dezactivează schimbările automate până când este selectat din nou modul automat.

Limba aplicației este și limba implicită a variantei de firmware. Pentru sistemele de scriere pe care un LCD1602/HD44780 standard nu le poate afișa în mod portabil, aplicația selectează varianta de firmware corespunzătoare cu text LCD în engleză; interfața desktop continuă să folosească limba selectată.

## Arduino și USB

Firmware-ul actual se află în `firmware/zapper_v5/zapper_v5.ino`, iar descrierea lui în `firmware/zapper_v5/README.md`. După încărcarea firmware-ului:

1. Deschideți vizualizarea **Dispozitiv**.
2. Selectați portul COM și faceți clic pe **Conectare**.
3. Așteptați starea **Gata**.
4. Trimiteți sesiunea de astăzi sau porniți o singură valoare în modul manual.
5. Verificați conexiunile de pe placă și apoi apăsați butonul fizic; ieșirea pornește abia după aceea.

Portul selectat este memorat în fișierul local `data/device.json`. Sesiunile de profil păstrează `device_steps` separate și exacte; o descriere precum „30 kHz” rămâne text lizibil pentru om, iar placa primește `30000000` milihertzi și durata în milisecunde.

### Limbile firmware-ului LCD

Firmware-ul 5.1.0 are 30 de variante lingvistice separate, generate dintr-o singură bază de cod. Fiecare sketch Arduino conține un singur set de texte LCD. Limbile care folosesc alfabetul latin au propriile texte scurte stocate ca ASCII sigur. Pentru chirilică și alte sisteme de scriere pe care un LCD1602/HD44780 obișnuit nu le poate afișa în mod portabil, varianta corespunzătoare folosește o interfață LCD în engleză. Lista completă se află în `firmware/LANGUAGES.md`.

Comanda `go run ./tools/firmware_i18n` generează toate sketch-urile în `build/generated/firmware/`. Procesul obișnuit `build.ps1` face acest lucru automat și include variantele în versiunea portable.

### Încărcarea firmware-ului din aplicație

Secțiunea **Dispozitiv → Firmware** afișează versiunea detectată, cea mai nouă versiune, limba variantei firmware și limba LCD. Utilizatorul alege bootloaderul nou sau vechi pentru Arduino Nano și face clic explicit pe **Încarcă firmware**; aplicația nu scrie niciodată firmware automat pe placă la pornire.

Compilarea și încărcarea sunt gestionate de `arduino-cli`. Zapper îl caută în `tools/arduino-cli/`, lângă EXE, în `PATH` și în locațiile obișnuite ale Arduino IDE. Dacă instrumentul nu este disponibil, aplicația indică clar acest lucru, iar butonul de încărcare rămâne dezactivat. Compilarea necesită și core-ul `arduino:avr` și biblioteca `LiquidCrystal_I2C` disponibile pentru instalarea `arduino-cli` utilizată.

### Detectarea limbii și alegerea firmware-ului

La pornire, aplicația citește limba mediului WebView2/Windows (`navigator.languages`) și o asociază cu unul dintre cele 30 de coduri acceptate. Dacă limba sistemului nu este acceptată, se selectează engleza. În modul **Automat (Windows)**, limba este verificată la fiecare pornire; o alegere manuală este memorată în `data/settings.json` până când modul automat este activat din nou.

Același cod de limbă este alegerea implicită pe ecranul de încărcare a firmware-ului. Pentru limbile pe care LCD1602 nu le acceptă, aplicația selectează tot varianta identificată prin limba utilizatorului, dar informează că textul LCD va fi în engleză. Firmware-ul nu este încărcat niciodată automat la pornirea aplicației; încărcarea necesită un clic explicit al utilizatorului pentru a nu suprascrie accidental un alt program deja stocat pe Arduino.

## Compilare

Este necesar Go. Cel mai simplu este să rulați în directorul rădăcină al proiectului:

```text
build.bat
```

Alternativ, în PowerShell:

```powershell
.\build.ps1
```

Scriptul rulează testele și analiza codului, construiește `build/generated/Zapper-dev.exe` și pregătește versiunea portable `build/Zapper/Zapper.exe` fără fereastră de consolă.

## Structura proiectului

- `app/` — cod Go, interfață HTML/CSS/JS, ghid și bază de date de frecvențe.
- `firmware/zapper_v5/` — firmware Arduino actual.
- `data/` — profiluri active, progres, arhivă, setări ale dispozitivului și copii de siguranță automate.
- `locales/` — traduceri versionate ale interfeței și ghidului, folosite în dezvoltare și copiate în versiunile publicate.
- `build/Zapper/` — versiune portable gata de copiat pe alt computer.