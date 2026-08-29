ZAPPER - WERSJA PORTABLE

Uruchom: Zapper.exe

Ten folder jest gotowy do przeniesienia na inny komputer.
Nie rozdzielaj Zapper.exe i folderu data, jeżeli chcesz zachować profile,
postęp, archiwum i ustawienia urządzenia.

Interfejs, SVG i baza częstotliwości są wbudowane w Zapper.exe.
Kompletne pakiety tłumaczeń UI i instrukcji są w folderze locales obok EXE.
Build jest przerywany, jeżeli któregokolwiek pakietu brakuje albo jest niepełny.
Aplikacja zapisuje dane w folderze data znajdującym się obok pliku EXE.
Wersja portable jest budowana bez prywatnych danych autora; potrzebne pliki
zostaną utworzone automatycznie przy pierwszym uruchomieniu.
Folder firmware zawiera aktualny firmware v5.1.0, 30 wariantów językowych LCD
oraz archiwalny kod v4.0. Warianty językowe są w firmware\localized.

Aplikacja wykrywa język Windows przy starcie (lub zapamiętuje wybór ręczny) i
proponuje odpowiadający mu wariant firmware. Wgrywanie nigdy nie rozpoczyna się
samo. Funkcja „Wgraj firmware” wymaga arduino-cli z core arduino:avr oraz
biblioteki LiquidCrystal_I2C. Zapper szuka arduino-cli także w tools\arduino-cli.

Zapper sprawdza przy uruchomieniu najnowszy GitHub Release. Jeżeli pojawi się
nowsza wersja, program zapyta przed pobraniem. Po zgodzie pobiera ZIP oraz plik
SHA-256, sprawdza sumę kontrolną, zachowuje cały folder data i uruchamia się
ponownie po aktualizacji. Kliknięcie numeru wersji w lewym dolnym rogu wymusza
ręczne sprawdzenie aktualizacji.

Na typowym Windows 10/11 wymagany WebView2 jest już obecny w systemie.
