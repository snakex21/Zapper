# Firmware Zapper v5.1.0

Firmware dla Arduino Nano ATmega328P współpracujący z aplikacją `zapper_go.exe`, ale zachowujący pełny tryb samodzielny z LCD i enkoderem. Wersja 5.1.0 dodaje osobne warianty językowe LCD bez zmiany logiki generatora ani protokołu sesji.

Domyślny plik `zapper_v5.ino` ma polski interfejs LCD. Zestaw 30 osobnych szkiców generuje `go run ./tools/firmware_i18n`; lista języków i zasady angielskiego fallbacku dla alfabetów nieobsługiwanych przez typowy LCD1602 znajdują się w `firmware/LANGUAGES.md`.

## Sprzęt i biblioteka

- LCD1602 16x2 I2C (niebieski), domyślny adres `0x27` (`LiquidCrystal_I2C`)
- enkoder: CLK `D2`, DT `D3`, SW `D4`
- wyjście sygnału: `D9 / OC1A`
- USB Serial: `115200`, 8 bitów, bez parzystości, 1 bit stopu

W Arduino IDE wybierz płytkę **Arduino Nano** i właściwy procesor/bootloader dla posiadanego egzemplarza, zainstaluj bibliotekę `LiquidCrystal_I2C`, a następnie wgraj `zapper_v5.ino`.

## Sterowanie bez komputera

- obrót wybiera pozycję lub zmienia wartość,
- krótki klik zatwierdza; w ręcznym ustawianiu częstotliwości zmienia krok regulacji,
- przytrzymanie w ręcznym ustawianiu przechodzi do czasu,
- klik podczas pracy natychmiast zatrzymuje wyjście,
- sesja przesłana z aplikacji czeka na potwierdzenie bez limitu czasu; na ekranie gotowości długie przytrzymanie ją anuluje,
- sesja wysłana przez USB najpierw pokazuje ekran `USB: SESJA GOTOWA`; dopiero krótki klik na płytce po sprawdzeniu uchwytów uruchamia wyjście,
- przytrzymanie przycisku na ekranie gotowości anuluje sesję; brak potwierdzenia nie ma limitu czasu,
- przytrzymanie w menu wygasza LCD; klik lub obrót budzi urządzenie.

## Protokół aplikacji

Częstotliwość jest przesyłana jako całkowita liczba miliherców, dzięki czemu nie ma problemu z separatorem dziesiętnym: `7.83 Hz = 7830`, `30 kHz = 30000000`.

```text
PING
CLEAR
STEP <częstotliwość_mHz> <czas_ms>
START
STOP
STATUS
```

Po `START` płytka odpowiada `ARMED <liczba_kroków>` i nie włącza D9. Po fizycznym potwierdzeniu przyciskiem wysyła `OK CONFIRMED`, a następnie rozpoczyna raportowanie `STATE RUNNING ...`. Dzięki temu samo polecenie z komputera nie uruchamia sygnału bez potwierdzenia przy płytce.

Sesja może zawierać maksymalnie 12 kroków. Zakres wynosi `0.1 Hz–4 MHz`, a czas jednego kroku `1 s–24 h`. Stan jest odsyłany co 500 ms. LCD1602 korzysta z bufora: firmware porównuje przygotowany obraz 16x2 z tym, co już jest na ekranie, i wysyła przez I2C wyłącznie zmienione znaki. Enkoder i przycisk są sprawdzane bez przerw w głównej pętli.

Jeżeli konkretny konwerter I2C ma adres `0x3F` zamiast `0x27`, zmień pierwszy parametr konstruktora `LiquidCrystal_I2C` w `zapper_v5.ino`. Samo okablowanie SDA/SCL pozostaje bez zmian.

Timer1 generuje falę sprzętowo na `D9`. Jedynie najniższy fragment zakresu `0.1–0.119 Hz`, którego 16-bitowy Timer1 nie obejmuje nawet z preskalerem 1024, korzysta z nieblokującego przełączania programowego.
