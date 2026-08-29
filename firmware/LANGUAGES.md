# Języki firmware LCD1602

Firmware Zapper v5.1.0 jest przygotowywany jako 30 osobnych szkiców Arduino. Każdy szkic zawiera tylko jeden zestaw napisów LCD, dzięki czemu Arduino Nano nie przechowuje tłumaczeń, których nie używa.

## Języki z własnym interfejsem LCD

W tych wariantach używany jest lokalny język zapisany bezpiecznym zestawem ASCII, tak aby działał na typowym LCD1602/HD44780 niezależnie od wariantu ROM znaków:

- `en` — English
- `pl` — Polski
- `de` — Deutsch
- `fr` — Français (na LCD bez znaków diakrytycznych)
- `es` — Español (na LCD bez znaków diakrytycznych)
- `it` — Italiano
- `pt` — Português (na LCD bez znaków diakrytycznych)
- `nl` — Nederlands
- `sv` — Svenska
- `no` — Norsk
- `da` — Dansk
- `fi` — Suomi
- `cs` — Čeština (na LCD bez znaków diakrytycznych)
- `sk` — Slovenčina (na LCD bez znaków diakrytycznych)
- `hu` — Magyar (na LCD bez znaków diakrytycznych)
- `ro` — Română (na LCD bez znaków diakrytycznych)
- `tr` — Türkçe (na LCD bez znaków diakrytycznych)
- `id` — Bahasa Indonesia
- `ms` — Bahasa Melayu
- `vi` — Tiếng Việt (na LCD bez znaków diakrytycznych)

## Warianty z angielskim fallbackiem

Typowy LCD1602 z kontrolerem HD44780 nie ma przenośnego zestawu znaków pozwalającego zagwarantować poprawne wyświetlanie poniższych alfabetów/pism na każdym module. Dlatego te warianty istnieją jako osobne firmware, ale ich LCD świadomie używa angielskich napisów:

- `ru` — Russian — cyrylica → English fallback
- `uk` — Ukrainian — cyrylica → English fallback
- `bg` — Bulgarian — cyrylica → English fallback
- `el` — Greek — alfabet grecki → English fallback
- `ar` — Arabic — pismo arabskie i RTL → English fallback
- `he` — Hebrew — pismo hebrajskie i RTL → English fallback
- `hi` — Hindi — dewanagari → English fallback
- `zh` — Chinese — znaki chińskie → English fallback
- `ja` — Japanese — brak gwarantowanej zgodności między wariantami ROM LCD1602 → English fallback
- `ko` — Korean — hangul → English fallback

Fallback dotyczy wyłącznie małego ekranu LCD1602 w urządzeniu. Aplikacja komputerowa może być tłumaczona normalnie na te języki, ponieważ nie ma ograniczeń zestawu znaków LCD.

## Generowanie 30 szkiców

Źródłem tłumaczeń jest `firmware/languages.json`, a głównym kodem `firmware/zapper_v5/zapper_v5.ino`. Polecenie:

```text
go run ./tools/firmware_i18n
```

tworzy osobne szkice w `build/generated/firmware/zapper_v5_<kod>/zapper_v5_<kod>.ino`.

Generator zastępuje wyłącznie blok napisów LCD i identyfikator języka. Logika generatora częstotliwości, obsługa Timer1, USB, przycisku, enkodera i zabezpieczeń pozostają identyczne we wszystkich 30 wariantach.
