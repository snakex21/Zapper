**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Die neue Version der Anwendung läuft in einem einzigen Fenster und benötigt weder Python noch Node.js oder Wails. Sie kann ohne angeschlossene Platine als Planer und Protokoll verwendet werden oder ein Arduino Nano über USB steuern.

## Lizenz und Verantwortung

Code, Firmware, Schaltpläne und Dokumentation sind für nichtkommerzielle Nutzung unter der Lizenz **PolyForm Noncommercial 1.0.0** öffentlich verfügbar. Sie dürfen im Rahmen dieser Lizenz verwendet, untersucht, verändert und weitergegeben werden, eine kommerzielle Nutzung des Projekts ist jedoch ohne gesonderte Zustimmung des Autors nicht erlaubt. Einzelheiten stehen in der Datei `LICENSE`.

Das Projekt wird ohne Gewährleistung für eigene Experimente und DIY-Anwendungen bereitgestellt. Der Benutzer ist für den korrekten Aufbau, Änderungen und die Art der Nutzung des Geräts verantwortlich. Der Autor haftet nicht für Hardwareschäden, sonstige Schäden oder Folgen eines falschen Aufbaus oder einer falschen Nutzung und garantiert keine bestimmten gesundheitlichen Wirkungen.

## Anwendung starten

Starte `Zapper.exe` aus dem Ordner der portablen Version. Dauerhafte Personen und ihre Kennungen werden in `data/persons.json` gespeichert, aktive Profile in `data/profiles.json`, und jeder Durchlauf besitzt eine eigene Datei in `data/progress/`. Abgeschlossene Durchläufe werden in Ordner `data/archive/<id>/` verschoben, die `profile.json` und `progress.json` enthalten. Die Einstellungen der Platine liegen in `data/device.json`, die Anwendungseinstellungen einschließlich der erkannten oder gewählten Sprache in `data/settings.json`. Sicherungen bleiben in lokalen Unterordnern `backups/`. Alles befindet sich neben der EXE; es wird nichts in AppData, Dokumente oder die Windows-Registrierung geschrieben.

In der Ansicht **Profile** können Personen hinzugefügt, fertiger Kontexttext für eine KI in die Zwischenablage erzeugt und vereinfachtes JSON aus einem KI-Modell eingefügt werden. Frequenzen werden in diesem Format als `frequency_hz` angegeben; die Anwendung prüft das Profil, zeigt eine Vorschau und erzeugt erst nach Bestätigung eine neue `run_id`. Der vorherige aktive Durchlauf dieser Person wird zuvor archiviert.

Während einer Profilsitzung speichert die Schaltfläche **Pause** den verbleibenden Teil des aktuellen Schritts und alle folgenden Schritte im lokalen Fortschritt. Beim Fortsetzen wird eine verkürzte Sequenz an die unveränderte Firmware gesendet und erneut eine physische Bestätigung an der Platine verlangt. **Stopp** verwirft den Teilfortschritt und lässt die vollständige Sitzung erneut ausführbar.

Übersprungene Sitzungen bleiben als überfällig in der Warteschlange. Die Programmregeln legen die Anzahl der Teile, Pausen innerhalb einer Serie, den Abstand zwischen vollständigen Sitzungen, die Abkühlzeit nach einer Sitzung sowie die Kompatibilität mit anderen Programmen am selben Tag fest. Ein Profil ohne überfällige Sitzungen wird nach Abschluss des Plans automatisch archiviert; mit **Programm beenden** kann es früher geschlossen werden.

## Anwendungssprache

Beim Start liest die Anwendung die Sprache von Windows/WebView2 und ordnet sie einer von 30 unterstützten Sprachen zu. Solange **Automatisch (Windows)** eingestellt ist, wird die Sprache bei jedem Start erneut erkannt. Eine manuelle Auswahl in der linken Seitenleiste wird in `data/settings.json` gespeichert und deaktiviert automatische Änderungen, bis der automatische Modus wieder gewählt wird.

Die Anwendungssprache ist zugleich die Standardsprache für die Firmware-Variante. Für Schriftsysteme, die ein gewöhnliches LCD1602/HD44780 nicht portabel darstellen kann, wählt die Anwendung die passende Firmware-Variante mit englischem LCD-Text; die Desktop-Oberfläche verwendet weiterhin die gewählte Sprache.

## Arduino und USB

Die aktuelle Firmware befindet sich unter `firmware/zapper_v5/zapper_v5.ino`, die Beschreibung unter `firmware/zapper_v5/README.md`. Nach dem Flashen der Firmware:

1. Öffne die Ansicht **Gerät**.
2. Wähle den COM-Port und klicke auf **Verbinden**.
3. Warte auf den Status **Bereit**.
4. Sende die heutige Sitzung oder starte im manuellen Modus einen einzelnen Wert.
5. Prüfe die Anschlüsse an der Platine und drücke anschließend die physische Taste; erst dann startet der Ausgang.

Der gewählte Port wird lokal in `data/device.json` gespeichert. Profilsitzungen enthalten eigene, genaue `device_steps`; eine Beschreibung wie „30 kHz“ bleibt menschenlesbarer Text, während die Platine `30000000` Millihertz und die Dauer in Millisekunden erhält.

### LCD-Firmware-Sprachen

Firmware 5.1.0 besitzt 30 separate Sprachvarianten, die aus einer gemeinsamen Codebasis erzeugt werden. Jeder Arduino-Sketch enthält nur einen Satz LCD-Texte. Sprachen mit lateinischem Alphabet besitzen eigene kurze, als sicheres ASCII gespeicherte Texte. Für Kyrillisch und andere Schriftsysteme, die ein typisches LCD1602/HD44780 nicht portabel anzeigen kann, verwendet die entsprechende Variante eine englische LCD-Oberfläche. Die vollständige Liste steht in `firmware/LANGUAGES.md`.

Der Befehl `go run ./tools/firmware_i18n` erzeugt alle Sketches in `build/generated/firmware/`. Der normale Ablauf über `build.ps1` erledigt dies automatisch und nimmt die Varianten in die portable Version auf.

### Firmware aus der Anwendung flashen

Der Bereich **Gerät → Firmware** zeigt die erkannte Version, die neueste Version, die Sprache der Firmware-Variante und die LCD-Sprache. Der Benutzer wählt den neuen oder alten Arduino-Nano-Bootloader und klickt ausdrücklich auf **Firmware flashen**; beim Start flasht die Anwendung die Platine niemals automatisch.

Kompilierung und Upload erfolgen über `arduino-cli`. Zapper sucht das Programm in `tools/arduino-cli/`, neben der EXE, in `PATH` und in typischen Arduino-IDE-Verzeichnissen. Ist das Werkzeug nicht vorhanden, zeigt die Anwendung dies klar an und die Flash-Schaltfläche bleibt deaktiviert. Für die Kompilierung müssen außerdem der Core `arduino:avr` und die Bibliothek `LiquidCrystal_I2C` für die verwendete `arduino-cli`-Installation verfügbar sein.

### Spracherkennung und Firmware-Auswahl

Beim Start liest die Anwendung die Sprache der WebView2-/Windows-Umgebung (`navigator.languages`) und ordnet sie einem der 30 unterstützten Codes zu. Wird die Systemsprache nicht unterstützt, wird Englisch gewählt. Im Modus **Automatisch (Windows)** erfolgt die Prüfung bei jedem Start; eine manuelle Auswahl wird in `data/settings.json` gespeichert, bis der automatische Modus wieder aktiviert wird.

Derselbe Sprachcode ist die Standardauswahl im Firmware-Flash-Bildschirm. Für Sprachen, die vom LCD1602 nicht unterstützt werden, wählt die Anwendung weiterhin die nach der Benutzersprache benannte Variante, weist aber darauf hin, dass der LCD-Text englisch ist. Firmware wird beim Start der Anwendung nie automatisch geflasht; das Flashen erfordert einen ausdrücklichen Klick des Benutzers, damit ein anderes bereits auf dem Arduino gespeichertes Programm nicht versehentlich überschrieben wird.

## Bauen

Go wird benötigt. Am einfachsten wird im Projektstamm ausgeführt:

```text
build.bat
```

Alternativ in PowerShell:

```powershell
.\build.ps1
```

Das Skript führt Tests und Codeanalyse aus, erstellt `build/generated/Zapper-dev.exe` und bereitet die portable `build/Zapper/Zapper.exe` ohne Konsolenfenster vor.

## Projektstruktur

- `app/` — Go-Code, HTML/CSS/JS-Oberfläche, Anleitung und Frequenzdatenbank.
- `firmware/zapper_v5/` — aktuelle Arduino-Firmware.
- `data/` — aktive Profile, Fortschritt, Archiv, Geräteeinstellungen und automatische Sicherungen.
- `locales/` — versionierte Übersetzungen für Oberfläche und Anleitung, die bei der Entwicklung genutzt und in Releases kopiert werden.
- `build/Zapper/` — fertige portable Version zum Kopieren auf einen anderen Computer.