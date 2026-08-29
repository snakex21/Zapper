**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Nowa wersja aplikacji działa w jednym oknie i nie wymaga Pythona, Node.js ani Wailsa. Może służyć jako plan i dziennik bez podłączonej płytki albo sterować Arduino Nano przez USB.

## Licencja i odpowiedzialność

Kod, firmware, schematy i dokumentacja są publicznie dostępne do użytku niekomercyjnego na licencji **PolyForm Noncommercial 1.0.0**. Wolno ich używać, analizować, modyfikować i rozpowszechniać w celach dozwolonych przez tę licencję, ale nie wolno wykorzystywać projektu komercyjnie bez osobnej zgody autora. Szczegóły znajdują się w pliku `LICENSE`.

Projekt jest udostępniany do samodzielnych eksperymentów i zastosowań DIY bez gwarancji. Użytkownik odpowiada za prawidłowy montaż, modyfikacje i sposób użycia urządzenia. Autor nie odpowiada za uszkodzenie sprzętu, inne szkody ani skutki nieprawidłowego montażu lub użycia i nie gwarantuje określonych efektów zdrowotnych.

## Uruchomienie

Uruchom `Zapper.exe` z folderu wersji portable. Stałe osoby i ich identyfikatory znajdują się w `data/persons.json`, aktywne profile w `data/profiles.json`, a każdy przebieg ma osobny plik w `data/progress/`. Zakończone przebiegi trafiają do folderów `data/archive/<id>/`, zawierających `profile.json` i `progress.json`. Ustawienia płytki są w `data/device.json`, a ustawienia aplikacji, w tym wykryty lub wybrany język, w `data/settings.json`. Kopie zapasowe pozostają w lokalnych podfolderach `backups/`. Całość znajduje się obok EXE — nic nie jest zapisywane w AppData, Dokumentach ani rejestrze Windows.

W widoku **Profile** można dodawać osoby, generować gotowy tekst kontekstu do schowka oraz wklejać uproszczony JSON otrzymany od AI. Częstotliwości w tym formacie podaje się jako `frequency_hz`; aplikacja waliduje profil, pokazuje podgląd i dopiero po potwierdzeniu tworzy nowy `run_id`. Poprzedni aktywny przebieg tej osoby trafia wcześniej do archiwum.

Podczas sesji profilu przycisk **Pauza** zapisuje pozostałą część bieżącego kroku i kolejnych kroków w lokalnym progresie. Wznowienie wysyła do niezmienionego firmware skróconą sekwencję i ponownie wymaga fizycznego potwierdzenia na płytce. **Zatrzymaj** anuluje częściowy postęp i pozostawia pełną sesję do ponownego wykonania.

Pominięte sesje pozostają w kolejce jako zaległe. Reguły programu określają liczbę części, przerwę wewnątrz serii, odstęp między pełnymi sesjami, cooldown po wykonaniu oraz wzajemną zgodność z innymi programami tego samego dnia. Profil bez zaległości trafia do archiwum automatycznie po ukończeniu planu, a przycisk **Zakończ program** pozwala zamknąć go wcześniej.

## Język aplikacji

Przy starcie aplikacja odczytuje język Windows/WebView2 i dopasowuje go do jednego z 30 obsługiwanych języków. Dopóki ustawienie ma tryb **Automatycznie (Windows)**, wykrywanie jest wykonywane przy każdym uruchomieniu. Ręczny wybór języka w lewym panelu jest zapamiętywany w `data/settings.json` i wyłącza automatyczną zmianę do chwili ponownego wybrania trybu automatycznego.

Język aplikacji jest również domyślnym językiem wariantu firmware. Dla alfabetów, których standardowy LCD1602/HD44780 nie wyświetla przenośnie, aplikacja wybiera odpowiedni wariant firmware z angielskim tekstem LCD; sam interfejs desktopowy nadal używa wybranego języka.

## Arduino i USB

Aktualny firmware znajduje się w `firmware/zapper_v5/zapper_v5.ino`, a opis w `firmware/zapper_v5/README.md`. Po wgraniu firmware:

1. Otwórz widok **Urządzenie**.
2. Wybierz port COM i kliknij **Połącz**.
3. Poczekaj na stan **Gotowe**.
4. Wyślij dzisiejszą sesję albo uruchom pojedynczą wartość w trybie ręcznym.
5. Na płytce sprawdź uchwyty, a następnie kliknij jej przycisk — dopiero wtedy wystartuje wyjście.

Wybrany port jest zapamiętywany w lokalnym `data/device.json`. Sesje profili przechowują osobne, dokładne `device_steps`; opis typu „30 kHz” pozostaje tekstem dla człowieka, a płytka dostaje `30000000` miliherców i czas w milisekundach.

### Języki firmware LCD

Firmware 5.1.0 ma 30 osobnych wariantów językowych generowanych z jednego kodu. Każdy szkic Arduino zawiera tylko jeden zestaw napisów LCD. Języki używające alfabetu łacińskiego mają własne krótkie napisy zapisane bezpiecznym ASCII. Dla cyrylicy oraz innych pism, których typowy LCD1602/HD44780 nie potrafi przenośnie wyświetlić, odpowiedni wariant używa angielskiego interfejsu LCD. Pełna lista jest w `firmware/LANGUAGES.md`.

Polecenie `go run ./tools/firmware_i18n` tworzy komplet szkiców w `build/generated/firmware/`. Zwykły `build.ps1` robi to automatycznie i dołącza warianty do wersji portable.

### Wgrywanie firmware z aplikacji

Widok **Urządzenie → Firmware** pokazuje wykrytą wersję, najnowszą wersję, język wariantu oraz język LCD. Użytkownik wybiera nowy lub stary bootloader Arduino Nano i sam klika **Wgraj firmware** — aplikacja nigdy nie flashuje płytki automatycznie przy uruchomieniu.

Do kompilacji i wgrania używany jest `arduino-cli`. Zapper szuka go w `tools/arduino-cli/`, obok EXE, w `PATH` oraz w typowych katalogach Arduino IDE. Jeżeli narzędzia nie ma, program pokazuje to wprost i przycisk wgrywania pozostaje nieaktywny. Do kompilacji potrzebny jest również core `arduino:avr` i biblioteka `LiquidCrystal_I2C` dostępne dla używanego `arduino-cli`.

### Wykrywanie języka i wybór firmware

Przy starcie aplikacja odczytuje język środowiska WebView2/Windows (`navigator.languages`) i dopasowuje go do jednego z 30 obsługiwanych kodów. Jeżeli systemowego języka nie ma na liście, wybierany jest angielski. W trybie **Automatycznie (Windows)** język jest sprawdzany przy każdym uruchomieniu; ręczny wybór jest zapamiętywany w `data/settings.json` do chwili ponownego włączenia trybu automatycznego.

Ten sam kod języka jest domyślnym wyborem dla ekranu wgrywania firmware. Dla języków, których LCD1602 nie obsługuje, aplikacja nadal wybiera wariant oznaczony językiem użytkownika, ale informuje, że napisy LCD będą angielskie. Firmware nie jest wgrywany automatycznie przy starcie aplikacji — wgrywanie wymaga świadomego kliknięcia użytkownika, aby przypadkiem nie nadpisać innego programu znajdującego się na Arduino.

## Budowanie

Wymagany jest Go. Najwygodniej uruchomić w głównym folderze:

```text
build.bat
```

Alternatywnie w PowerShell:

```powershell
.\build.ps1
```

Skrypt wykonuje testy, analizę kodu, buduje `build/generated/Zapper-dev.exe` oraz przygotowuje portable `build/Zapper/Zapper.exe` bez okna konsoli.

## Układ projektu

- `app/` — kod Go, interfejs HTML/CSS/JS, instrukcja i baza częstotliwości.
- `firmware/zapper_v5/` — aktualny firmware Arduino.
- `data/` — aktywne profile, postęp, archiwum, ustawienia urządzenia i automatyczne kopie zapasowe.
- `locales/` — wersjonowane tłumaczenia interfejsu i instrukcji używane podczas pracy oraz kopiowane do wydań.
- `build/Zapper/` — gotowa wersja portable do skopiowania na drugi komputer.