**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Versi baharu aplikasi berjalan dalam satu tetingkap dan tidak memerlukan Python, Node.js atau Wails. Ia boleh digunakan sebagai perancang dan log tanpa papan yang disambungkan, atau untuk mengawal Arduino Nano melalui USB.

## Lesen dan tanggungjawab

Kod, firmware, skema dan dokumentasi tersedia secara terbuka untuk kegunaan bukan komersial di bawah lesen **PolyForm Noncommercial 1.0.0**. Semuanya boleh digunakan, dikaji, diubah suai dan diedarkan bagi tujuan yang dibenarkan oleh lesen tersebut, tetapi projek ini tidak boleh digunakan secara komersial tanpa kebenaran berasingan daripada pengarang. Lihat fail `LICENSE` untuk butiran.

Projek ini disediakan tanpa jaminan untuk eksperimen sendiri dan kegunaan DIY. Pengguna bertanggungjawab terhadap pemasangan yang betul, pengubahsuaian dan cara peranti digunakan. Pengarang tidak bertanggungjawab atas kerosakan perkakasan, kerugian lain atau akibat pemasangan atau penggunaan yang salah dan tidak menjamin sebarang kesan kesihatan tertentu.

## Menjalankan aplikasi

Jalankan `Zapper.exe` dari folder versi portable. Orang yang disimpan secara kekal dan pengecam mereka berada dalam `data/persons.json`, profil aktif dalam `data/profiles.json`, dan setiap larian mempunyai fail sendiri dalam `data/progress/`. Larian yang selesai dipindahkan ke folder `data/archive/<id>/` yang mengandungi `profile.json` dan `progress.json`. Tetapan papan disimpan dalam `data/device.json`, manakala tetapan aplikasi termasuk bahasa yang dikesan atau dipilih disimpan dalam `data/settings.json`. Sandaran kekal dalam subfolder tempatan `backups/`. Semuanya berada di sebelah EXE; tiada apa-apa ditulis ke AppData, Documents atau Windows Registry.

Dalam paparan **Profil**, anda boleh menambah orang, menghasilkan teks konteks AI yang sedia disalin ke papan klip dan menampal JSON ringkas yang dikembalikan oleh model AI. Frekuensi dalam format ini diberikan sebagai `frequency_hz`; aplikasi mengesahkan profil, menunjukkan pratonton dan hanya mencipta `run_id` baharu selepas pengesahan. Larian aktif sebelumnya bagi orang tersebut diarkibkan terlebih dahulu.

Semasa sesi profil, butang **Jeda** menyimpan baki bahagian langkah semasa dan semua langkah seterusnya dalam kemajuan tempatan. Apabila diteruskan, urutan yang dipendekkan dihantar kepada firmware yang tidak berubah dan pengesahan fizikal pada papan diperlukan semula. **Henti** membatalkan kemajuan separa dan membiarkan keseluruhan sesi tersedia untuk dijalankan semula.

Sesi yang dilangkau kekal dalam barisan sebagai tertunggak. Peraturan program menentukan bilangan bahagian, rehat di dalam satu siri, jarak antara sesi penuh, tempoh pemulihan selepas sesi dan keserasian dengan program lain pada hari yang sama. Profil tanpa sesi tertunggak diarkibkan secara automatik selepas rancangan selesai, manakala **Tamatkan program** membolehkan ia ditutup lebih awal.

## Bahasa aplikasi

Semasa aplikasi bermula, bahasa Windows/WebView2 dibaca dan dipadankan dengan salah satu daripada 30 bahasa yang disokong. Selagi tetapan berada dalam mod **Automatik (Windows)**, pengesanan bahasa dilakukan pada setiap pelancaran. Pilihan bahasa secara manual pada panel kiri disimpan dalam `data/settings.json` dan melumpuhkan perubahan automatik sehingga mod automatik dipilih semula.

Bahasa aplikasi juga menjadi bahasa lalai bagi varian firmware. Untuk sistem tulisan yang tidak dapat dipaparkan secara mudah alih oleh LCD1602/HD44780 standard, aplikasi memilih varian firmware yang sepadan dengan teks LCD bahasa Inggeris; antara muka desktop masih menggunakan bahasa yang dipilih.

## Arduino dan USB

Firmware semasa berada di `firmware/zapper_v5/zapper_v5.ino`, dan penerangannya di `firmware/zapper_v5/README.md`. Selepas firmware dimuat naik:

1. Buka paparan **Peranti**.
2. Pilih port COM dan klik **Sambung**.
3. Tunggu keadaan **Sedia**.
4. Hantar sesi hari ini atau mulakan satu nilai dalam mod manual.
5. Periksa sambungan pada papan, kemudian tekan butang fizikalnya; output hanya bermula selepas itu.

Port yang dipilih diingati dalam fail tempatan `data/device.json`. Sesi profil menyimpan `device_steps` yang berasingan dan tepat; penerangan seperti “30 kHz” kekal sebagai teks yang mudah dibaca, manakala papan menerima `30000000` millihertz dan tempoh dalam milisaat.

### Bahasa firmware LCD

Firmware 5.1.0 mempunyai 30 varian bahasa berasingan yang dibina daripada satu asas kod. Setiap sketch Arduino hanya mengandungi satu set teks LCD. Bahasa yang menggunakan abjad Latin mempunyai teks ringkas sendiri yang disimpan sebagai ASCII selamat. Untuk Cyrillic dan sistem tulisan lain yang tidak dapat dipaparkan secara mudah alih oleh LCD1602/HD44780 biasa, varian berkaitan menggunakan antara muka LCD bahasa Inggeris. Senarai penuh tersedia dalam `firmware/LANGUAGES.md`.

Perintah `go run ./tools/firmware_i18n` membina semua sketch dalam `build/generated/firmware/`. Proses biasa `build.ps1` melakukan ini secara automatik dan memasukkan varian tersebut ke dalam versi portable.

### Memuat naik firmware dari aplikasi

Bahagian **Peranti → Firmware** menunjukkan versi yang dikesan, versi terkini, bahasa varian firmware dan bahasa LCD. Pengguna memilih bootloader Arduino Nano baharu atau lama dan secara jelas mengklik **Muat naik firmware**; aplikasi tidak pernah menulis firmware ke papan secara automatik semasa permulaan.

Kompilasi dan muat naik dikendalikan oleh `arduino-cli`. Zapper mencarinya dalam `tools/arduino-cli/`, di sebelah EXE, dalam `PATH` dan di lokasi Arduino IDE yang biasa. Jika alat tersebut tiada, aplikasi menyatakannya dengan jelas dan butang muat naik kekal dilumpuhkan. Kompilasi juga memerlukan core `arduino:avr` dan pustaka `LiquidCrystal_I2C` tersedia untuk pemasangan `arduino-cli` yang digunakan.

### Pengesanan bahasa dan pemilihan firmware

Semasa permulaan, aplikasi membaca bahasa persekitaran WebView2/Windows (`navigator.languages`) dan memadankannya dengan salah satu daripada 30 kod yang disokong. Jika bahasa sistem tidak disokong, bahasa Inggeris dipilih. Dalam mod **Automatik (Windows)**, bahasa diperiksa pada setiap pelancaran; pilihan manual disimpan dalam `data/settings.json` sehingga mod automatik diaktifkan semula.

Kod bahasa yang sama ialah pilihan lalai pada skrin muat naik firmware. Bagi bahasa yang tidak disokong oleh LCD1602, aplikasi masih memilih varian yang ditandai dengan bahasa pengguna tetapi memaklumkan bahawa teks LCD akan menggunakan bahasa Inggeris. Firmware tidak pernah dimuat naik secara automatik apabila aplikasi bermula; muat naik memerlukan klik jelas daripada pengguna supaya program lain yang sudah disimpan pada Arduino tidak ditindih secara tidak sengaja.

## Membina

Go diperlukan. Cara paling mudah ialah menjalankan ini di folder akar projek:

```text
build.bat
```

Sebagai alternatif, dalam PowerShell:

```powershell
.\build.ps1
```

Skrip menjalankan ujian dan analisis kod, membina `build/generated/Zapper-dev.exe` dan menyediakan portable `build/Zapper/Zapper.exe` tanpa tetingkap konsol.

## Struktur projek

- `app/` — kod Go, antara muka HTML/CSS/JS, panduan dan pangkalan data frekuensi.
- `firmware/zapper_v5/` — firmware Arduino semasa.
- `data/` — profil aktif, kemajuan, arkib, tetapan peranti dan sandaran automatik.
- `locales/` — terjemahan antara muka dan panduan yang disimpan dalam kawalan versi, digunakan semasa pembangunan dan disalin ke keluaran.
- `build/Zapper/` — versi portable siap disalin ke komputer lain.