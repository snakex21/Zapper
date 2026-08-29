**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Az alkalmazás új verziója egyetlen ablakban fut, és nem igényel Pythont, Node.js-t vagy Wailst. Használható tervezőként és naplóként csatlakoztatott panel nélkül, vagy egy Arduino Nano USB-n keresztüli vezérlésére.

## Licenc és felelősség

A kód, a firmware, a kapcsolási rajzok és a dokumentáció nem kereskedelmi használatra nyilvánosan elérhető a **PolyForm Noncommercial 1.0.0** licenc alatt. A licenc által engedélyezett célokra használhatók, tanulmányozhatók, módosíthatók és terjeszthetők, de a projekt kereskedelmi felhasználása a szerző külön engedélye nélkül nem megengedett. A részleteket a `LICENSE` fájl tartalmazza.

A projekt garancia nélkül, önálló kísérletekhez és DIY használathoz kerül közzétételre. A felhasználó felel a helyes összeszerelésért, a módosításokért és az eszköz használatának módjáért. A szerző nem vállal felelősséget hardverkárért, egyéb károkért vagy a hibás összeszerelésből vagy használatból eredő következményekért, és nem garantál konkrét egészségügyi hatásokat.

## Az alkalmazás indítása

Indítsd el a `Zapper.exe` fájlt a hordozható verzió mappájából. Az állandó személyek és azonosítóik a `data/persons.json` fájlban, az aktív profilok a `data/profiles.json` fájlban találhatók, és minden futás külön fájlt kap a `data/progress/` mappában. A befejezett futások a `data/archive/<id>/` mappákba kerülnek, amelyek `profile.json` és `progress.json` fájlokat tartalmaznak. A panel beállításai a `data/device.json`, az alkalmazás beállításai, beleértve az érzékelt vagy kiválasztott nyelvet is, a `data/settings.json` fájlban vannak. A biztonsági mentések a helyi `backups/` almappákban maradnak. Minden az EXE mellett található; semmi nem kerül az AppData, Dokumentumok mappába vagy a Windows rendszerleíró adatbázisába.

A **Profilok** nézetben személyeket adhatsz hozzá, kész AI-környezetszöveget generálhatsz a vágólapra, és beillesztheted egy AI-modell által visszaadott egyszerűsített JSON-t. A frekvenciákat ebben a formátumban `frequency_hz` mezőként kell megadni; az alkalmazás ellenőrzi a profilt, előnézetet mutat, és csak jóváhagyás után hoz létre új `run_id` értéket. Az adott személy korábbi aktív futása előbb az archívumba kerül.

Profilfuttatás közben a **Szünet** gomb elmenti az aktuális lépés fennmaradó részét és az összes következő lépést a helyi előrehaladásba. A folytatás rövidített sorozatot küld a változatlan firmware-nek, és ismét fizikai megerősítést kér a panelen. A **Leállítás** törli a részleges előrehaladást, és a teljes munkamenetet újbóli végrehajtásra hagyja.

A kihagyott munkamenetek késedelmesként a sorban maradnak. A programszabályok meghatározzák a részek számát, a sorozaton belüli szüneteket, a teljes munkamenetek közötti távolságot, a munkamenet utáni pihenőidőt és az ugyanazon a napon futó más programokkal való kompatibilitást. A késedelmes munkamenet nélküli profil a terv befejezése után automatikusan archiválódik, míg a **Program befejezése** lehetővé teszi a korábbi lezárást.

## Az alkalmazás nyelve

Indításkor az alkalmazás beolvassa a Windows/WebView2 nyelvét, és hozzárendeli a 30 támogatott nyelv egyikéhez. Amíg a beállítás **Automatikus (Windows)** módban van, a nyelvfelismerés minden indításkor megtörténik. A bal oldali panelen végzett kézi nyelvválasztás a `data/settings.json` fájlba kerül, és letiltja az automatikus váltást addig, amíg újra az automatikus módot választod.

Az alkalmazás nyelve egyben a firmware-változat alapértelmezett nyelve is. Azoknál az írásrendszereknél, amelyeket egy szabványos LCD1602/HD44780 nem tud hordozható módon megjeleníteni, az alkalmazás a megfelelő firmware-változatot választja angol LCD-szöveggel; az asztali felület továbbra is a kiválasztott nyelvet használja.

## Arduino és USB

Az aktuális firmware a `firmware/zapper_v5/zapper_v5.ino` fájlban, a leírás pedig a `firmware/zapper_v5/README.md` fájlban található. A firmware feltöltése után:

1. Nyisd meg az **Eszköz** nézetet.
2. Válaszd ki a COM-portot, és kattints a **Csatlakozás** gombra.
3. Várd meg a **Kész** állapotot.
4. Küldd el a mai munkamenetet, vagy indíts egyetlen értéket kézi módban.
5. Ellenőrizd a panel csatlakozásait, majd nyomd meg a fizikai gombot; a kimenet csak ezután indul el.

A kiválasztott portot a helyi `data/device.json` fájl jegyzi meg. A profil-munkamenetek külön, pontos `device_steps` lépéseket tárolnak; egy olyan leírás, mint a „30 kHz”, ember számára olvasható szöveg marad, miközben a panel `30000000` millihertzet és az időt milliszekundumban kapja meg.

### LCD firmware nyelvek

A 5.1.0 firmware 30 külön nyelvi változattal rendelkezik, amelyek egyetlen kódbázisból készülnek. Minden Arduino sketch csak egy LCD-szövegkészletet tartalmaz. A latin ábécét használó nyelvek saját rövid, biztonságos ASCII-ként tárolt szövegeket kapnak. A cirill és más olyan írásrendszerek esetében, amelyeket egy tipikus LCD1602/HD44780 nem tud megbízhatóan megjeleníteni, a megfelelő változat angol LCD-felületet használ. A teljes lista a `firmware/LANGUAGES.md` fájlban található.

A `go run ./tools/firmware_i18n` parancs minden sketchet a `build/generated/firmware/` mappába hoz létre. A szokásos `build.ps1` folyamat ezt automatikusan elvégzi, és a változatokat belefoglalja a hordozható verzióba.

### Firmware feltöltése az alkalmazásból

Az **Eszköz → Firmware** rész mutatja az észlelt verziót, a legújabb verziót, a firmware-változat nyelvét és az LCD nyelvét. A felhasználó kiválasztja az Arduino Nano új vagy régi bootloaderét, majd kifejezetten a **Firmware feltöltése** gombra kattint; az alkalmazás indításkor soha nem ír firmware-t automatikusan a panelre.

A fordítást és feltöltést az `arduino-cli` végzi. A Zapper a `tools/arduino-cli/` mappában, az EXE mellett, a `PATH` változóban és a szokásos Arduino IDE helyeken keresi. Ha az eszköz nem érhető el, az alkalmazás ezt egyértelműen jelzi, és a feltöltési gomb inaktív marad. A fordításhoz az `arduino:avr` core és a `LiquidCrystal_I2C` könyvtár is szükséges az adott `arduino-cli` telepítésben.

### Nyelvfelismerés és firmware-választás

Indításkor az alkalmazás beolvassa a WebView2/Windows környezet nyelvét (`navigator.languages`), és hozzárendeli a 30 támogatott kód egyikéhez. Ha a rendszer nyelve nem támogatott, az angol kerül kiválasztásra. **Automatikus (Windows)** módban a nyelvet minden indításkor ellenőrzi; a kézi választás a `data/settings.json` fájlban marad, amíg az automatikus módot újra be nem kapcsolod.

Ugyanez a nyelvkód az alapértelmezett választás a firmware feltöltési képernyőjén. Az LCD1602 által nem támogatott nyelveknél az alkalmazás továbbra is a felhasználó nyelvével jelölt változatot választja, de jelzi, hogy az LCD szövege angol lesz. A firmware az alkalmazás indításakor soha nem töltődik fel automatikusan; a feltöltéshez a felhasználó kifejezett kattintása szükséges, hogy egy másik, már az Arduinón lévő program véletlenül ne íródjon felül.

## Fordítás és build

A Go szükséges. A legegyszerűbb a projekt gyökérmappájában futtatni:

```text
build.bat
```

Vagy PowerShellben:

```powershell
.\build.ps1
```

A szkript teszteket és kódelemzést futtat, elkészíti a `build/generated/Zapper-dev.exe` fájlt, és előkészíti a hordozható `build/Zapper/Zapper.exe` fájlt konzolablak nélkül.

## Projekt felépítése

- `app/` — Go-kód, HTML/CSS/JS felület, útmutató és frekvencia-adatbázis.
- `firmware/zapper_v5/` — aktuális Arduino firmware.
- `data/` — aktív profilok, előrehaladás, archívum, eszközbeállítások és automatikus biztonsági mentések.
- `locales/` — verziókezelt felület- és útmutatófordítások, amelyeket fejlesztéskor használunk és a kiadásokba másolunk.
- `build/Zapper/` — kész hordozható verzió, amely másik számítógépre másolható.