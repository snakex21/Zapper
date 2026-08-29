**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Uygulamanın yeni sürümü tek bir pencerede çalışır ve Python, Node.js veya Wails gerektirmez. Bağlı bir kart olmadan planlayıcı ve kayıt defteri olarak kullanılabilir ya da Arduino Nano'yu USB üzerinden kontrol edebilir.

## Lisans ve sorumluluk

Kod, firmware, bağlantı şemaları ve belgeler **PolyForm Noncommercial 1.0.0** lisansı kapsamında ticari olmayan kullanım için herkese açıktır. Bu lisansın izin verdiği amaçlarla kullanılabilir, incelenebilir, değiştirilebilir ve dağıtılabilir; ancak proje, yazarın ayrıca izni olmadan ticari amaçla kullanılamaz. Ayrıntılar `LICENSE` dosyasındadır.

Proje, kişisel deneyler ve DIY kullanımı için garanti verilmeksizin sunulur. Doğru montajdan, değişikliklerden ve cihazın kullanım biçiminden kullanıcı sorumludur. Yazar; donanım hasarından, diğer zararlardan veya yanlış montaj ya da kullanımdan doğan sonuçlardan sorumlu değildir ve belirli sağlık etkilerini garanti etmez.

## Uygulamayı çalıştırma

Portable sürüm klasöründeki `Zapper.exe` dosyasını çalıştırın. Kalıcı kişiler ve kimlikleri `data/persons.json` içinde, etkin profiller `data/profiles.json` içinde saklanır ve her çalışma için `data/progress/` altında ayrı bir dosya bulunur. Tamamlanan çalışmalar `profile.json` ve `progress.json` içeren `data/archive/<id>/` klasörlerine taşınır. Kart ayarları `data/device.json`, algılanan veya seçilen dil dâhil uygulama ayarları ise `data/settings.json` içinde saklanır. Yedekler yerel `backups/` alt klasörlerinde kalır. Her şey EXE dosyasının yanındadır; AppData, Belgeler veya Windows Kayıt Defteri'ne hiçbir şey yazılmaz.

**Profiller** görünümünde kişi ekleyebilir, panoya kopyalanmaya hazır AI bağlam metni oluşturabilir ve bir AI modelinin döndürdüğü basitleştirilmiş JSON'u yapıştırabilirsiniz. Bu biçimde frekanslar `frequency_hz` olarak verilir; uygulama profili doğrular, önizleme gösterir ve yalnızca onaydan sonra yeni bir `run_id` oluşturur. Kişinin önceki etkin çalışması önce arşivlenir.

Bir profil oturumu sırasında **Duraklat** düğmesi, geçerli adımın kalan kısmını ve sonraki tüm adımları yerel ilerlemeye kaydeder. Devam ettirme, değişmemiş firmware'e kısaltılmış bir sıra gönderir ve kart üzerinde yeniden fiziksel onay ister. **Durdur**, kısmi ilerlemeyi iptal eder ve tüm oturumu yeniden çalıştırılabilir durumda bırakır.

Atlanan oturumlar gecikmiş olarak kuyrukta kalır. Program kuralları; parça sayısını, seri içindeki araları, tam oturumlar arasındaki mesafeyi, oturum sonrası bekleme süresini ve aynı gün diğer programlarla uyumluluğu belirler. Gecikmiş oturumu olmayan bir profil plan tamamlandığında otomatik olarak arşivlenir; **Programı bitir** ise daha erken kapatmaya izin verir.

## Uygulama dili

Uygulama başlatılırken Windows/WebView2 dili okunur ve desteklenen 30 dilden biriyle eşleştirilir. Ayar **Otomatik (Windows)** modunda kaldığı sürece dil algılama her başlatmada yapılır. Sol panelde elle seçilen dil `data/settings.json` içine kaydedilir ve otomatik mod yeniden seçilene kadar otomatik değişiklikleri devre dışı bırakır.

Uygulama dili aynı zamanda firmware varyantının varsayılan dilidir. Standart LCD1602/HD44780'in taşınabilir biçimde gösteremediği yazı sistemlerinde uygulama, İngilizce LCD metni kullanan karşılık gelen firmware varyantını seçer; masaüstü arayüzü yine seçilen dili kullanır.

## Arduino ve USB

Güncel firmware `firmware/zapper_v5/zapper_v5.ino` dosyasındadır; açıklaması `firmware/zapper_v5/README.md` içindedir. Firmware yüklendikten sonra:

1. **Cihaz** görünümünü açın.
2. COM portunu seçin ve **Bağlan** düğmesine tıklayın.
3. **Hazır** durumunu bekleyin.
4. Bugünkü oturumu gönderin veya manuel modda tek bir değer başlatın.
5. Kart üzerindeki bağlantıları kontrol edin ve ardından fiziksel düğmeye basın; çıkış ancak bundan sonra başlar.

Seçilen port yerel `data/device.json` dosyasında hatırlanır. Profil oturumları ayrı ve kesin `device_steps` değerlerini saklar; “30 kHz” gibi bir açıklama insanın okuyabileceği metin olarak kalırken kart `30000000` milihertz ve süreyi milisaniye olarak alır.

### LCD firmware dilleri

Firmware 5.1.0, tek bir kod tabanından oluşturulan 30 ayrı dil varyantına sahiptir. Her Arduino sketch yalnızca bir LCD metin kümesi içerir. Latin alfabesini kullanan dillerin güvenli ASCII olarak saklanan kendi kısa metinleri vardır. Tipik LCD1602/HD44780'in taşınabilir biçimde gösteremediği Kiril ve diğer yazı sistemlerinde ilgili varyant İngilizce LCD arayüzü kullanır. Tam liste `firmware/LANGUAGES.md` dosyasındadır.

`go run ./tools/firmware_i18n` komutu tüm sketch'leri `build/generated/firmware/` altında oluşturur. Normal `build.ps1` süreci bunu otomatik yapar ve varyantları portable sürüme ekler.

### Uygulamadan firmware yükleme

**Cihaz → Firmware** bölümü algılanan sürümü, en yeni sürümü, firmware varyantının dilini ve LCD dilini gösterir. Kullanıcı Arduino Nano için yeni veya eski bootloader'ı seçer ve açıkça **Firmware yükle** düğmesine tıklar; uygulama başlangıçta karta hiçbir zaman otomatik firmware yazmaz.

Derleme ve yükleme `arduino-cli` tarafından yapılır. Zapper bu aracı `tools/arduino-cli/` içinde, EXE'nin yanında, `PATH` içinde ve tipik Arduino IDE konumlarında arar. Araç yoksa uygulama bunu açıkça bildirir ve yükleme düğmesi devre dışı kalır. Derleme için kullanılan `arduino-cli` kurulumunda ayrıca `arduino:avr` core'u ve `LiquidCrystal_I2C` kitaplığı bulunmalıdır.

### Dil algılama ve firmware seçimi

Başlangıçta uygulama WebView2/Windows ortam dilini (`navigator.languages`) okur ve desteklenen 30 koddan biriyle eşleştirir. Sistem dili desteklenmiyorsa İngilizce seçilir. **Otomatik (Windows)** modunda dil her başlatmada kontrol edilir; manuel seçim, otomatik mod yeniden etkinleştirilene kadar `data/settings.json` içinde saklanır.

Aynı dil kodu firmware yükleme ekranında varsayılan seçimdir. LCD1602'nin desteklemediği diller için uygulama yine kullanıcının diliyle işaretlenmiş varyantı seçer ancak LCD metninin İngilizce olacağını bildirir. Uygulama başlatılırken firmware hiçbir zaman otomatik yüklenmez; Arduino'da zaten bulunan başka bir programın yanlışlıkla üzerine yazılmaması için yükleme kullanıcının açık bir tıklamasını gerektirir.

## Derleme

Go gereklidir. En kolay yol proje kökünde şunu çalıştırmaktır:

```text
build.bat
```

Alternatif olarak PowerShell'de:

```powershell
.\build.ps1
```

Betik testleri ve kod analizini çalıştırır, `build/generated/Zapper-dev.exe` dosyasını oluşturur ve konsol penceresi olmadan portable `build/Zapper/Zapper.exe` dosyasını hazırlar.

## Proje yapısı

- `app/` — Go kodu, HTML/CSS/JS arayüzü, kılavuz ve frekans veritabanı.
- `firmware/zapper_v5/` — güncel Arduino firmware'i.
- `data/` — etkin profiller, ilerleme, arşiv, cihaz ayarları ve otomatik yedekler.
- `locales/` — geliştirme sırasında kullanılan ve sürümlere kopyalanan, sürüm denetimindeki arayüz ve kılavuz çevirileri.
- `build/Zapper/` — başka bir bilgisayara kopyalanmaya hazır portable sürüm.