**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

La nuova versione dell’applicazione funziona in un’unica finestra e non richiede Python, Node.js o Wails. Può essere usata come pianificatore e registro senza una scheda collegata, oppure può controllare un Arduino Nano tramite USB.

## Licenza e responsabilità

Codice, firmware, schemi e documentazione sono disponibili pubblicamente per uso non commerciale con licenza **PolyForm Noncommercial 1.0.0**. Possono essere usati, studiati, modificati e distribuiti per gli scopi consentiti dalla licenza, ma il progetto non può essere utilizzato commercialmente senza un’autorizzazione separata dell’autore. Per i dettagli consulta il file `LICENSE`.

Il progetto è fornito senza garanzia per esperimenti indipendenti e utilizzi DIY. L’utente è responsabile del corretto montaggio, delle modifiche e del modo in cui il dispositivo viene utilizzato. L’autore non è responsabile di danni all’hardware, altre perdite o conseguenze di un montaggio o uso non corretto e non garantisce particolari effetti sulla salute.

## Avvio dell’applicazione

Avvia `Zapper.exe` dalla cartella della versione portable. Le persone persistenti e i relativi identificatori sono salvati in `data/persons.json`, i profili attivi in `data/profiles.json` e ogni esecuzione ha un proprio file in `data/progress/`. Le esecuzioni concluse vengono spostate nelle cartelle `data/archive/<id>/`, che contengono `profile.json` e `progress.json`. Le impostazioni della scheda sono salvate in `data/device.json`, mentre quelle dell’applicazione, compresa la lingua rilevata o selezionata, sono in `data/settings.json`. I backup rimangono nelle sottocartelle locali `backups/`. Tutto si trova accanto all’EXE; nulla viene scritto in AppData, Documenti o nel Registro di Windows.

Nella vista **Profili** puoi aggiungere persone, generare testo di contesto per l’AI pronto da copiare e incollare JSON semplificato restituito da un modello AI. Le frequenze in questo formato vengono fornite come `frequency_hz`; l’applicazione convalida il profilo, mostra un’anteprima e crea un nuovo `run_id` solo dopo la conferma. L’esecuzione attiva precedente della persona viene prima archiviata.

Durante una sessione del profilo, il pulsante **Pausa** salva la parte rimanente del passaggio corrente e tutti i passaggi successivi nel progresso locale. La ripresa invia una sequenza abbreviata al firmware invariato e richiede nuovamente una conferma fisica sulla scheda. **Ferma** annulla il progresso parziale e lascia l’intera sessione disponibile per essere eseguita di nuovo.

Le sessioni saltate rimangono in coda come scadute. Le regole del programma definiscono il numero di parti, le pause all’interno di una serie, l’intervallo tra sessioni complete, il tempo di recupero dopo una sessione e la compatibilità con altri programmi nello stesso giorno. Un profilo senza sessioni scadute viene archiviato automaticamente al termine del piano, mentre **Termina programma** consente di chiuderlo prima.

## Lingua dell’applicazione

All’avvio l’applicazione legge la lingua di Windows/WebView2 e la associa a una delle 30 lingue supportate. Finché l’impostazione rimane in modalità **Automatico (Windows)**, il rilevamento viene eseguito a ogni avvio. Una scelta manuale nel pannello sinistro viene salvata in `data/settings.json` e disabilita le modifiche automatiche finché non viene selezionata di nuovo la modalità automatica.

La lingua dell’applicazione è anche la lingua predefinita della variante firmware. Per i sistemi di scrittura che un normale LCD1602/HD44780 non può visualizzare in modo portabile, l’applicazione seleziona la variante firmware corrispondente con testo LCD in inglese; l’interfaccia desktop continua comunque a usare la lingua selezionata.

## Arduino e USB

Il firmware corrente si trova in `firmware/zapper_v5/zapper_v5.ino`, con la descrizione in `firmware/zapper_v5/README.md`. Dopo aver caricato il firmware:

1. Apri la vista **Dispositivo**.
2. Seleziona la porta COM e fai clic su **Connetti**.
3. Attendi lo stato **Pronto**.
4. Invia la sessione di oggi oppure avvia un singolo valore in modalità manuale.
5. Controlla i collegamenti sulla scheda e poi premi il pulsante fisico; solo allora partirà l’uscita.

La porta selezionata viene memorizzata nel file locale `data/device.json`. Le sessioni del profilo conservano `device_steps` separati e precisi; una descrizione come “30 kHz” resta testo leggibile, mentre la scheda riceve `30000000` millihertz e la durata in millisecondi.

### Lingue del firmware LCD

Il firmware 5.1.0 dispone di 30 varianti linguistiche separate generate da un’unica base di codice. Ogni sketch Arduino contiene un solo set di testi LCD. Le lingue che usano l’alfabeto latino hanno brevi testi propri memorizzati in ASCII sicuro. Per il cirillico e altri sistemi di scrittura che un tipico LCD1602/HD44780 non può visualizzare in modo portabile, la variante corrispondente usa un’interfaccia LCD in inglese. L’elenco completo è disponibile in `firmware/LANGUAGES.md`.

Il comando `go run ./tools/firmware_i18n` genera tutti gli sketch in `build/generated/firmware/`. Il normale processo `build.ps1` lo fa automaticamente e include le varianti nella versione portable.

### Caricamento del firmware dall’applicazione

La sezione **Dispositivo → Firmware** mostra la versione rilevata, l’ultima versione, la lingua della variante firmware e la lingua LCD. L’utente seleziona il bootloader nuovo o vecchio dell’Arduino Nano e fa esplicitamente clic su **Carica firmware**; l’applicazione non esegue mai il flash automatico della scheda all’avvio.

La compilazione e il caricamento sono gestiti da `arduino-cli`. Zapper lo cerca in `tools/arduino-cli/`, accanto all’EXE, in `PATH` e nelle posizioni tipiche di Arduino IDE. Se lo strumento non è disponibile, l’applicazione lo indica chiaramente e il pulsante di caricamento rimane disabilitato. La compilazione richiede anche il core `arduino:avr` e la libreria `LiquidCrystal_I2C` disponibili per l’installazione di `arduino-cli` utilizzata.

### Rilevamento della lingua e scelta del firmware

All’avvio l’applicazione legge la lingua dell’ambiente WebView2/Windows (`navigator.languages`) e la associa a uno dei 30 codici supportati. Se la lingua di sistema non è supportata, viene selezionato l’inglese. In modalità **Automatico (Windows)** la lingua viene controllata a ogni avvio; una selezione manuale viene salvata in `data/settings.json` finché non viene riattivata la modalità automatica.

Lo stesso codice lingua è la scelta predefinita nella schermata di caricamento del firmware. Per le lingue non supportate da LCD1602, l’applicazione seleziona comunque la variante identificata dalla lingua dell’utente, ma informa che il testo LCD sarà in inglese. Il firmware non viene mai caricato automaticamente all’avvio dell’applicazione; il caricamento richiede un clic esplicito dell’utente per evitare di sovrascrivere accidentalmente un altro programma già presente su Arduino.

## Compilazione

È richiesto Go. Il modo più semplice è eseguire nella cartella principale del progetto:

```text
build.bat
```

In alternativa, in PowerShell:

```powershell
.\build.ps1
```

Lo script esegue test e analisi del codice, crea `build/generated/Zapper-dev.exe` e prepara il portable `build/Zapper/Zapper.exe` senza finestra della console.

## Struttura del progetto

- `app/` — codice Go, interfaccia HTML/CSS/JS, guida e database delle frequenze.
- `firmware/zapper_v5/` — firmware Arduino corrente.
- `data/` — profili attivi, progresso, archivio, impostazioni del dispositivo e backup automatici.
- `locales/` — traduzioni versionate dell’interfaccia e della guida, usate nello sviluppo e copiate nelle release.
- `build/Zapper/` — versione portable pronta da copiare su un altro computer.