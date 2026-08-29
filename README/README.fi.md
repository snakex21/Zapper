**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Sovelluksen uusi versio toimii yhdessä ikkunassa eikä vaadi Pythonia, Node.js:ää tai Wailsia. Sitä voi käyttää suunnittelijana ja lokina ilman kytkettyä piirilevyä tai Arduino Nanon ohjaamiseen USB:n kautta.

## Lisenssi ja vastuu

Koodi, laiteohjelmisto, kytkentäkaaviot ja dokumentaatio ovat julkisesti saatavilla ei-kaupalliseen käyttöön **PolyForm Noncommercial 1.0.0** -lisenssillä. Niitä saa käyttää, tutkia, muokata ja jakaa lisenssin sallimiin tarkoituksiin, mutta projektia ei saa käyttää kaupallisesti ilman tekijän erillistä lupaa. Tarkemmat tiedot ovat tiedostossa `LICENSE`.

Projekti tarjotaan ilman takuuta omiin kokeiluihin ja DIY-käyttöön. Käyttäjä vastaa oikeasta kokoamisesta, muutoksista ja laitteen käyttötavasta. Tekijä ei vastaa laitteistovaurioista, muista vahingoista tai virheellisen kokoamisen tai käytön seurauksista eikä takaa tiettyjä terveysvaikutuksia.

## Sovelluksen käynnistäminen

Käynnistä `Zapper.exe` portable-version kansiosta. Pysyvät henkilöt ja heidän tunnisteensa tallennetaan tiedostoon `data/persons.json`, aktiiviset profiilit tiedostoon `data/profiles.json`, ja jokaisella ajolla on oma tiedostonsa kansiossa `data/progress/`. Valmiit ajot siirretään kansioihin `data/archive/<id>/`, joissa ovat `profile.json` ja `progress.json`. Piirilevyn asetukset tallennetaan tiedostoon `data/device.json`, ja sovelluksen asetukset, mukaan lukien tunnistettu tai valittu kieli, tiedostoon `data/settings.json`. Varmuuskopiot jäävät paikallisiin `backups/`-alikansioihin. Kaikki sijaitsee EXE-tiedoston vieressä; mitään ei kirjoiteta AppDataan, Asiakirjat-kansioon tai Windowsin rekisteriin.

**Profiilit**-näkymässä voit lisätä henkilöitä, luoda valmiin AI-kontekstitekstin leikepöydälle ja liittää AI-mallin palauttamaa yksinkertaistettua JSONia. Taajuudet annetaan tässä muodossa kentällä `frequency_hz`; sovellus tarkistaa profiilin, näyttää esikatselun ja luo uuden `run_id`-tunnisteen vasta vahvistuksen jälkeen. Henkilön aiempi aktiivinen ajo arkistoidaan ensin.

Profiili-istunnon aikana **Tauko**-painike tallentaa nykyisen vaiheen jäljellä olevan osan ja kaikki seuraavat vaiheet paikalliseen etenemiseen. Jatkaminen lähettää lyhennetyn sekvenssin muuttamattomalle laiteohjelmistolle ja vaatii jälleen fyysisen vahvistuksen piirilevyllä. **Pysäytä** peruuttaa osittaisen etenemisen ja jättää koko istunnon suoritettavaksi uudelleen.

Ohitetut istunnot jäävät jonoon myöhästyneinä. Ohjelman säännöt määrittävät osien määrän, sarjan sisäiset tauot, täysien istuntojen välisen ajan, istunnon jälkeisen palautumisajan sekä yhteensopivuuden muiden saman päivän ohjelmien kanssa. Profiili, jolla ei ole myöhästyneitä istuntoja, arkistoidaan automaattisesti suunnitelman valmistuttua; **Lopeta ohjelma** mahdollistaa sen sulkemisen aikaisemmin.

## Sovelluksen kieli

Käynnistyksen yhteydessä sovellus lukee Windows/WebView2-ympäristön kielen ja yhdistää sen yhteen 30 tuetusta kielestä. Niin kauan kuin asetuksena on **Automaattinen (Windows)**, kieli tunnistetaan jokaisella käynnistyksellä. Vasemman paneelin manuaalinen kielivalinta tallennetaan tiedostoon `data/settings.json` ja poistaa automaattiset muutokset käytöstä, kunnes automaattinen tila valitaan uudelleen.

Sovelluksen kieli on myös laiteohjelmistoversion oletuskieli. Kirjoitusjärjestelmille, joita tavallinen LCD1602/HD44780 ei pysty näyttämään luotettavasti, sovellus valitsee vastaavan laiteohjelmistoversion englanninkielisillä LCD-teksteillä; työpöytäkäyttöliittymä käyttää edelleen valittua kieltä.

## Arduino ja USB

Nykyinen laiteohjelmisto on tiedostossa `firmware/zapper_v5/zapper_v5.ino`, ja sen kuvaus tiedostossa `firmware/zapper_v5/README.md`. Kun laiteohjelmisto on ladattu:

1. Avaa **Laite**-näkymä.
2. Valitse COM-portti ja napsauta **Yhdistä**.
3. Odota tilaa **Valmis**.
4. Lähetä tämän päivän istunto tai käynnistä yksi arvo manuaalitilassa.
5. Tarkista piirilevyn liitännät ja paina sitten fyysistä painiketta; lähtö käynnistyy vasta tämän jälkeen.

Valittu portti muistetaan paikallisessa tiedostossa `data/device.json`. Profiili-istunnot tallentavat erilliset ja tarkat `device_steps`-vaiheet; esimerkiksi “30 kHz” säilyy ihmiselle luettavana tekstinä, kun taas piirilevy vastaanottaa arvon `30000000` millihertseinä sekä ajan millisekunteina.

### LCD-laiteohjelmiston kielet

Laiteohjelmistossa 5.1.0 on 30 erillistä kieliversiota, jotka muodostetaan yhdestä koodipohjasta. Jokainen Arduino-sketch sisältää vain yhden LCD-tekstisarjan. Latinalaista aakkostoa käyttävillä kielillä on omat lyhyet tekstinsä turvallisena ASCII-muodossa. Kyrillisille ja muille kirjoitusjärjestelmille, joita tavallinen LCD1602/HD44780 ei pysty näyttämään luotettavasti, vastaava versio käyttää englanninkielistä LCD-käyttöliittymää. Täydellinen luettelo on tiedostossa `firmware/LANGUAGES.md`.

Komento `go run ./tools/firmware_i18n` luo kaikki sketchit kansioon `build/generated/firmware/`. Normaali `build.ps1` tekee tämän automaattisesti ja sisällyttää versiot portable-pakettiin.

### Laiteohjelmiston lataaminen sovelluksesta

Kohta **Laite → Firmware** näyttää tunnistetun version, uusimman version, laiteohjelmistoversion kielen ja LCD-kielen. Käyttäjä valitsee Arduino Nanon uuden tai vanhan bootloaderin ja napsauttaa nimenomaisesti **Lataa firmware**; sovellus ei koskaan kirjoita laiteohjelmistoa automaattisesti käynnistyksen yhteydessä.

Kääntäminen ja lataaminen tehdään `arduino-cli`-työkalulla. Zapper etsii sitä kansiosta `tools/arduino-cli/`, EXE-tiedoston vierestä, `PATH`-ympäristömuuttujasta ja tavallisista Arduino IDE -sijainneista. Jos työkalua ei ole saatavilla, sovellus ilmoittaa siitä selvästi ja latauspainike pysyy poissa käytöstä. Kääntäminen vaatii myös `arduino:avr`-ytimen ja `LiquidCrystal_I2C`-kirjaston käytettävälle `arduino-cli`-asennukselle.

### Kielen tunnistus ja firmware-version valinta

Käynnistyksen yhteydessä sovellus lukee WebView2/Windows-ympäristön kielen (`navigator.languages`) ja yhdistää sen yhteen 30 tuetusta kielikoodista. Jos järjestelmän kieltä ei tueta, valitaan englanti. Tilassa **Automaattinen (Windows)** kieli tarkistetaan jokaisella käynnistyksellä; manuaalinen valinta tallennetaan tiedostoon `data/settings.json`, kunnes automaattinen tila otetaan uudelleen käyttöön.

Sama kielikoodi on oletusvalinta firmware-latausnäytössä. Jos LCD1602 ei tue valittua kieltä, sovellus valitsee silti käyttäjän kielellä nimetyn version mutta ilmoittaa, että LCD-tekstit ovat englanniksi. Laiteohjelmistoa ei koskaan ladata automaattisesti sovelluksen käynnistyessä; lataaminen vaatii käyttäjän nimenomaisen napsautuksen, jotta Arduinoon jo tallennettua muuta ohjelmaa ei vahingossa korvata.

## Kääntäminen

Go vaaditaan. Helpoin tapa on suorittaa projektin juurikansiossa:

```text
build.bat
```

Vaihtoehtoisesti PowerShellissä:

```powershell
.\build.ps1
```

Skripti suorittaa testit ja koodianalyysin, rakentaa `build/generated/Zapper-dev.exe`-tiedoston ja valmistelee portable-version `build/Zapper/Zapper.exe` ilman konsoli-ikkunaa.

## Projektin rakenne

- `app/` — Go-koodi, HTML/CSS/JS-käyttöliittymä, ohje ja taajuustietokanta.
- `firmware/zapper_v5/` — nykyinen Arduino-laiteohjelmisto.
- `data/` — aktiiviset profiilit, eteneminen, arkisto, laiteasetukset ja automaattiset varmuuskopiot.
- `locales/` — versionhallinnassa olevat käyttöliittymän ja oppaan käännökset, joita käytetään kehityksessä ja kopioidaan julkaisuihin.
- `build/Zapper/` — valmis portable-versio toiselle tietokoneelle kopioitavaksi.