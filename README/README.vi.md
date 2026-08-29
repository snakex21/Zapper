**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

Phiên bản mới của ứng dụng chạy trong một cửa sổ duy nhất và không cần Python, Node.js hay Wails. Ứng dụng có thể dùng như công cụ lập kế hoạch và nhật ký khi chưa kết nối bo mạch, hoặc điều khiển Arduino Nano qua USB.

## Giấy phép và trách nhiệm

Mã nguồn, firmware, sơ đồ và tài liệu được công khai cho mục đích phi thương mại theo giấy phép **PolyForm Noncommercial 1.0.0**. Chúng có thể được sử dụng, nghiên cứu, sửa đổi và phân phối cho các mục đích mà giấy phép cho phép, nhưng dự án không được sử dụng cho mục đích thương mại nếu chưa có sự cho phép riêng của tác giả. Xem tệp `LICENSE` để biết chi tiết.

Dự án được cung cấp không kèm bảo hành cho các thử nghiệm độc lập và mục đích DIY. Người dùng chịu trách nhiệm về việc lắp ráp đúng, các sửa đổi và cách sử dụng thiết bị. Tác giả không chịu trách nhiệm về hư hỏng phần cứng, các thiệt hại khác hoặc hậu quả do lắp ráp hay sử dụng sai, và không đảm bảo bất kỳ hiệu quả sức khỏe cụ thể nào.

## Chạy ứng dụng

Chạy `Zapper.exe` từ thư mục của bản portable. Danh sách người dùng cố định và mã định danh của họ được lưu trong `data/persons.json`, các hồ sơ đang hoạt động trong `data/profiles.json`, và mỗi lần chạy có tệp riêng trong `data/progress/`. Các lần chạy đã hoàn thành được chuyển vào các thư mục `data/archive/<id>/` chứa `profile.json` và `progress.json`. Cài đặt bo mạch được lưu trong `data/device.json`, còn cài đặt ứng dụng, bao gồm ngôn ngữ được phát hiện hoặc lựa chọn, nằm trong `data/settings.json`. Bản sao lưu vẫn nằm trong các thư mục con cục bộ `backups/`. Mọi thứ nằm cạnh tệp EXE; không có gì được ghi vào AppData, Documents hay Windows Registry.

Trong màn hình **Hồ sơ**, bạn có thể thêm người, tạo văn bản ngữ cảnh cho AI sẵn sàng sao chép vào clipboard và dán JSON đơn giản do mô hình AI trả về. Tần số trong định dạng này được cung cấp bằng `frequency_hz`; ứng dụng xác thực hồ sơ, hiển thị bản xem trước và chỉ tạo `run_id` mới sau khi xác nhận. Lần chạy đang hoạt động trước đó của người này sẽ được lưu trữ trước.

Trong một phiên của hồ sơ, nút **Tạm dừng** lưu phần còn lại của bước hiện tại và tất cả các bước tiếp theo vào tiến trình cục bộ. Khi tiếp tục, ứng dụng gửi một chuỗi rút gọn tới firmware không thay đổi và lại yêu cầu xác nhận vật lý trên bo mạch. **Dừng** hủy tiến trình một phần và để toàn bộ phiên sẵn sàng chạy lại.

Các phiên bị bỏ qua vẫn nằm trong hàng đợi với trạng thái quá hạn. Quy tắc chương trình xác định số phần, thời gian nghỉ trong một chuỗi, khoảng cách giữa các phiên đầy đủ, thời gian nghỉ sau một phiên và khả năng tương thích với các chương trình khác trong cùng ngày. Một hồ sơ không còn phiên quá hạn sẽ được tự động lưu trữ khi kế hoạch hoàn tất, còn **Kết thúc chương trình** cho phép đóng hồ sơ sớm hơn.

## Ngôn ngữ ứng dụng

Khi khởi động, ứng dụng đọc ngôn ngữ Windows/WebView2 và ánh xạ ngôn ngữ đó tới một trong 30 ngôn ngữ được hỗ trợ. Chừng nào cài đặt vẫn ở chế độ **Tự động (Windows)**, việc phát hiện ngôn ngữ sẽ được thực hiện mỗi lần khởi động. Lựa chọn ngôn ngữ thủ công trong bảng bên trái được lưu vào `data/settings.json` và tắt thay đổi tự động cho đến khi chế độ tự động được chọn lại.

Ngôn ngữ ứng dụng cũng là ngôn ngữ mặc định của biến thể firmware. Với các hệ chữ mà LCD1602/HD44780 tiêu chuẩn không thể hiển thị một cách ổn định, ứng dụng chọn biến thể firmware tương ứng với văn bản LCD bằng tiếng Anh; giao diện desktop vẫn dùng ngôn ngữ đã chọn.

## Arduino và USB

Firmware hiện tại nằm trong `firmware/zapper_v5/zapper_v5.ino`, phần mô tả nằm trong `firmware/zapper_v5/README.md`. Sau khi nạp firmware:

1. Mở màn hình **Thiết bị**.
2. Chọn cổng COM và nhấn **Kết nối**.
3. Chờ trạng thái **Sẵn sàng**.
4. Gửi phiên của hôm nay hoặc khởi động một giá trị đơn trong chế độ thủ công.
5. Kiểm tra các kết nối trên bo mạch rồi nhấn nút vật lý; đầu ra chỉ bắt đầu sau bước này.

Cổng đã chọn được ghi nhớ trong tệp cục bộ `data/device.json`. Các phiên hồ sơ lưu các `device_steps` riêng biệt và chính xác; mô tả như “30 kHz” vẫn là văn bản dễ đọc, trong khi bo mạch nhận `30000000` millihertz và thời lượng tính bằng mili giây.

### Ngôn ngữ firmware LCD

Firmware 5.1.0 có 30 biến thể ngôn ngữ riêng biệt được tạo từ một cơ sở mã duy nhất. Mỗi Arduino sketch chỉ chứa một bộ văn bản LCD. Các ngôn ngữ dùng bảng chữ cái Latin có các chuỗi ngắn riêng được lưu dưới dạng ASCII an toàn. Với chữ Cyrillic và các hệ chữ khác mà LCD1602/HD44780 thông thường không thể hiển thị ổn định, biến thể tương ứng sử dụng giao diện LCD bằng tiếng Anh. Danh sách đầy đủ nằm trong `firmware/LANGUAGES.md`.

Lệnh `go run ./tools/firmware_i18n` tạo toàn bộ sketch trong `build/generated/firmware/`. Quy trình `build.ps1` thông thường thực hiện việc này tự động và đưa các biến thể vào bản portable.

### Nạp firmware từ ứng dụng

Phần **Thiết bị → Firmware** hiển thị phiên bản được phát hiện, phiên bản mới nhất, ngôn ngữ của biến thể firmware và ngôn ngữ LCD. Người dùng chọn bootloader Arduino Nano mới hoặc cũ rồi chủ động nhấn **Nạp firmware**; ứng dụng không bao giờ tự động ghi firmware lên bo mạch khi khởi động.

Việc biên dịch và tải lên được thực hiện bằng `arduino-cli`. Zapper tìm công cụ này trong `tools/arduino-cli/`, cạnh EXE, trong `PATH` và tại các vị trí Arduino IDE phổ biến. Nếu công cụ không có, ứng dụng thông báo rõ ràng và nút nạp firmware vẫn bị vô hiệu hóa. Việc biên dịch cũng yêu cầu core `arduino:avr` và thư viện `LiquidCrystal_I2C` có sẵn cho bản cài đặt `arduino-cli` đang dùng.

### Phát hiện ngôn ngữ và chọn firmware

Khi khởi động, ứng dụng đọc ngôn ngữ môi trường WebView2/Windows (`navigator.languages`) và ánh xạ tới một trong 30 mã được hỗ trợ. Nếu ngôn ngữ hệ thống không được hỗ trợ, tiếng Anh sẽ được chọn. Trong chế độ **Tự động (Windows)**, ngôn ngữ được kiểm tra ở mỗi lần khởi động; lựa chọn thủ công được lưu trong `data/settings.json` cho đến khi chế độ tự động được bật lại.

Cùng mã ngôn ngữ đó là lựa chọn mặc định trên màn hình nạp firmware. Với các ngôn ngữ mà LCD1602 không hỗ trợ, ứng dụng vẫn chọn biến thể mang mã ngôn ngữ của người dùng nhưng thông báo rằng văn bản LCD sẽ bằng tiếng Anh. Firmware không bao giờ được nạp tự động khi ứng dụng khởi động; việc nạp yêu cầu một cú nhấp rõ ràng từ người dùng để tránh vô tình ghi đè chương trình khác đã có trên Arduino.

## Biên dịch

Cần cài Go. Cách đơn giản nhất là chạy trong thư mục gốc của dự án:

```text
build.bat
```

Hoặc trong PowerShell:

```powershell
.\build.ps1
```

Script chạy kiểm thử và phân tích mã, tạo `build/generated/Zapper-dev.exe` và chuẩn bị bản portable `build/Zapper/Zapper.exe` không có cửa sổ console.

## Cấu trúc dự án

- `app/` — mã Go, giao diện HTML/CSS/JS, hướng dẫn và cơ sở dữ liệu tần số.
- `firmware/zapper_v5/` — firmware Arduino hiện tại.
- `data/` — hồ sơ đang hoạt động, tiến trình, kho lưu trữ, cài đặt thiết bị và bản sao lưu tự động.
- `locales/` — các bản dịch giao diện và hướng dẫn được quản lý phiên bản, dùng khi phát triển và sao chép vào bản phát hành.
- `build/Zapper/` — phiên bản portable hoàn chỉnh để sao chép sang máy tính khác.