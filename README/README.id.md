**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Versi baru aplikasi berjalan dalam satu jendela dan tidak memerlukan Python, Node.js, atau Wails. Aplikasi dapat digunakan sebagai perencana dan catatan tanpa papan yang terhubung, atau untuk mengendalikan Arduino Nano melalui USB.

## Lisensi dan tanggung jawab

Kode, firmware, skema, dan dokumentasi tersedia untuk umum bagi penggunaan nonkomersial di bawah lisensi **PolyForm Noncommercial 1.0.0**. Semuanya boleh digunakan, dipelajari, diubah, dan didistribusikan untuk tujuan yang diizinkan oleh lisensi tersebut, tetapi proyek tidak boleh digunakan secara komersial tanpa izin terpisah dari pembuatnya. Lihat berkas `LICENSE` untuk rinciannya.

Proyek ini disediakan tanpa jaminan untuk eksperimen mandiri dan penggunaan DIY. Pengguna bertanggung jawab atas perakitan yang benar, modifikasi, dan cara perangkat digunakan. Pembuat tidak bertanggung jawab atas kerusakan perangkat keras, kerugian lain, atau akibat dari perakitan maupun penggunaan yang salah, dan tidak menjamin efek kesehatan tertentu.

## Menjalankan aplikasi

Jalankan `Zapper.exe` dari folder versi portable. Data orang yang tetap dan pengenalnya disimpan di `data/persons.json`, profil aktif di `data/profiles.json`, dan setiap run memiliki berkas tersendiri di `data/progress/`. Run yang selesai dipindahkan ke folder `data/archive/<id>/` yang berisi `profile.json` dan `progress.json`. Pengaturan papan disimpan di `data/device.json`, sedangkan pengaturan aplikasi, termasuk bahasa yang terdeteksi atau dipilih, disimpan di `data/settings.json`. Cadangan tetap berada di subfolder lokal `backups/`. Semuanya berada di samping EXE; tidak ada yang ditulis ke AppData, Documents, atau Windows Registry.

Di tampilan **Profil**, Anda dapat menambahkan orang, membuat teks konteks AI yang siap disalin ke clipboard, dan menempelkan JSON sederhana yang dikembalikan oleh model AI. Frekuensi dalam format ini diberikan sebagai `frequency_hz`; aplikasi memvalidasi profil, menampilkan pratinjau, dan hanya membuat `run_id` baru setelah konfirmasi. Run aktif sebelumnya milik orang tersebut diarsipkan terlebih dahulu.

Selama sesi profil, tombol **Jeda** menyimpan sisa bagian dari langkah saat ini dan semua langkah berikutnya ke progres lokal. Saat dilanjutkan, aplikasi mengirim urutan yang dipersingkat ke firmware yang tidak berubah dan kembali meminta konfirmasi fisik pada papan. **Hentikan** membatalkan progres sebagian dan membiarkan seluruh sesi tersedia untuk dijalankan lagi.

Sesi yang dilewati tetap berada dalam antrean sebagai sesi terlambat. Aturan program menentukan jumlah bagian, jeda di dalam satu rangkaian, jarak antar sesi penuh, waktu pemulihan setelah sesi, serta kompatibilitas dengan program lain pada hari yang sama. Profil tanpa sesi terlambat diarsipkan otomatis setelah rencana selesai, sedangkan **Akhiri program** memungkinkan profil ditutup lebih awal.

## Bahasa aplikasi

Saat aplikasi dimulai, bahasa Windows/WebView2 dibaca dan dipetakan ke salah satu dari 30 bahasa yang didukung. Selama pengaturan berada dalam mode **Otomatis (Windows)**, deteksi bahasa dilakukan setiap kali aplikasi dijalankan. Pilihan bahasa manual pada panel kiri disimpan di `data/settings.json` dan menonaktifkan perubahan otomatis sampai mode otomatis dipilih kembali.

Bahasa aplikasi juga menjadi bahasa bawaan untuk varian firmware. Untuk sistem tulisan yang tidak dapat ditampilkan secara portabel oleh LCD1602/HD44780 standar, aplikasi memilih varian firmware yang sesuai dengan teks LCD berbahasa Inggris; antarmuka desktop tetap menggunakan bahasa yang dipilih.

## Arduino dan USB

Firmware saat ini berada di `firmware/zapper_v5/zapper_v5.ino`, dengan keterangannya di `firmware/zapper_v5/README.md`. Setelah firmware diunggah:

1. Buka tampilan **Perangkat**.
2. Pilih port COM lalu klik **Hubungkan**.
3. Tunggu sampai status **Siap**.
4. Kirim sesi hari ini atau mulai satu nilai dalam mode manual.
5. Periksa sambungan pada papan, lalu tekan tombol fisiknya; keluaran baru dimulai setelah itu.

Port yang dipilih diingat dalam berkas lokal `data/device.json`. Sesi profil menyimpan `device_steps` yang terpisah dan presisi; deskripsi seperti “30 kHz” tetap menjadi teks yang mudah dibaca, sementara papan menerima `30000000` millihertz dan durasi dalam milidetik.

### Bahasa firmware LCD

Firmware 5.1.0 memiliki 30 varian bahasa terpisah yang dibuat dari satu basis kode. Setiap sketch Arduino hanya memuat satu set teks LCD. Bahasa yang memakai alfabet Latin memiliki teks singkat masing-masing yang disimpan sebagai ASCII aman. Untuk Cyrillic dan sistem tulisan lain yang tidak dapat ditampilkan secara portabel oleh LCD1602/HD44780 biasa, varian terkait menggunakan antarmuka LCD berbahasa Inggris. Daftar lengkap tersedia di `firmware/LANGUAGES.md`.

Perintah `go run ./tools/firmware_i18n` membuat semua sketch di `build/generated/firmware/`. Proses `build.ps1` biasa melakukan hal ini secara otomatis dan memasukkan semua varian ke versi portable.

### Mengunggah firmware dari aplikasi

Bagian **Perangkat → Firmware** menampilkan versi yang terdeteksi, versi terbaru, bahasa varian firmware, dan bahasa LCD. Pengguna memilih bootloader Arduino Nano baru atau lama lalu secara eksplisit mengklik **Unggah firmware**; aplikasi tidak pernah menulis firmware ke papan secara otomatis saat startup.

Kompilasi dan pengunggahan ditangani oleh `arduino-cli`. Zapper mencarinya di `tools/arduino-cli/`, di samping EXE, di `PATH`, dan di lokasi Arduino IDE yang umum. Jika alat tersebut tidak tersedia, aplikasi menampilkannya dengan jelas dan tombol unggah tetap dinonaktifkan. Kompilasi juga memerlukan core `arduino:avr` dan library `LiquidCrystal_I2C` tersedia untuk instalasi `arduino-cli` yang digunakan.

### Deteksi bahasa dan pemilihan firmware

Saat startup, aplikasi membaca bahasa lingkungan WebView2/Windows (`navigator.languages`) dan memetakannya ke salah satu dari 30 kode yang didukung. Jika bahasa sistem tidak didukung, bahasa Inggris dipilih. Dalam mode **Otomatis (Windows)**, bahasa diperiksa setiap kali aplikasi dijalankan; pilihan manual disimpan di `data/settings.json` sampai mode otomatis diaktifkan kembali.

Kode bahasa yang sama menjadi pilihan bawaan di layar pengunggahan firmware. Untuk bahasa yang tidak didukung LCD1602, aplikasi tetap memilih varian yang ditandai dengan bahasa pengguna, tetapi memberi tahu bahwa teks LCD akan berbahasa Inggris. Firmware tidak pernah diunggah otomatis ketika aplikasi dimulai; pengunggahan memerlukan klik eksplisit dari pengguna agar program lain yang sudah tersimpan pada Arduino tidak tertimpa secara tidak sengaja.

## Membangun

Go diperlukan. Cara paling mudah adalah menjalankan ini di folder utama proyek:

```text
build.bat
```

Atau melalui PowerShell:

```powershell
.\build.ps1
```

Skrip menjalankan pengujian dan analisis kode, membangun `build/generated/Zapper-dev.exe`, lalu menyiapkan portable `build/Zapper/Zapper.exe` tanpa jendela konsol.

## Struktur proyek

- `app/` — kode Go, antarmuka HTML/CSS/JS, panduan, dan basis data frekuensi.
- `firmware/zapper_v5/` — firmware Arduino saat ini.
- `data/` — profil aktif, progres, arsip, pengaturan perangkat, dan cadangan otomatis.
- `locales/` — terjemahan antarmuka dan panduan yang disimpan dalam version control, dipakai saat pengembangan dan disalin ke rilis.
- `build/Zapper/` — versi portable siap disalin ke komputer lain.