**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

新版应用在单个窗口中运行，不需要 Python、Node.js 或 Wails。它可以在未连接开发板时作为计划与记录工具使用，也可以通过 USB 控制 Arduino Nano。

## 许可证与责任

代码、firmware、电路图和文档依据 **PolyForm Noncommercial 1.0.0** 许可证公开提供，用于非商业用途。可以在该许可证允许的范围内使用、研究、修改和分发，但未经作者另行许可，不得将本项目用于商业用途。详细内容请参阅 `LICENSE` 文件。

本项目以无担保方式提供，用于独立实验和 DIY 使用。用户负责正确组装、修改以及设备的使用方式。作者不对硬件损坏、其他损失或因错误组装或使用造成的后果承担责任，也不保证任何特定的健康效果。

## 运行应用

从 portable 版本文件夹运行 `Zapper.exe`。固定人员及其标识符保存在 `data/persons.json`，活动配置保存在 `data/profiles.json`，每次运行在 `data/progress/` 中都有独立文件。已完成的运行会移动到 `data/archive/<id>/` 文件夹，其中包含 `profile.json` 和 `progress.json`。开发板设置保存在 `data/device.json`，应用设置（包括检测到或手动选择的语言）保存在 `data/settings.json`。备份保留在本地 `backups/` 子文件夹中。所有数据都位于 EXE 文件旁边；不会写入 AppData、Documents 或 Windows 注册表。

在 **配置** 视图中，可以添加人员、生成可直接复制到剪贴板的 AI 上下文文本，并粘贴 AI 模型返回的简化 JSON。该格式中的频率使用 `frequency_hz`；应用会验证配置、显示预览，并且仅在确认后创建新的 `run_id`。该人员之前的活动运行会先被归档。

在配置会话期间，**暂停** 按钮会把当前步骤剩余部分以及之后所有步骤保存到本地进度中。继续时会向未修改的 firmware 发送缩短后的序列，并再次要求在开发板上进行物理确认。**停止** 会取消部分进度，并保留完整会话供重新执行。

被跳过的会话会以逾期状态保留在队列中。程序规则定义分段数量、系列内部的休息时间、完整会话之间的间隔、会话后的冷却时间，以及与同一天其他程序的兼容性。没有逾期会话的配置会在计划完成后自动归档，而 **结束程序** 可以提前关闭它。

## 应用语言

应用启动时会读取 Windows/WebView2 的语言，并映射到 30 种受支持语言之一。只要设置保持在 **自动（Windows）** 模式，每次启动都会重新检测语言。左侧面板中的手动语言选择会保存到 `data/settings.json`，并停止自动切换，直到重新选择自动模式。

应用语言同时也是 firmware 变体的默认语言。对于标准 LCD1602/HD44780 无法可靠显示的文字系统，应用会选择对应的 firmware 变体，并在 LCD 上使用英文文本；桌面界面仍然使用所选语言。

## Arduino 与 USB

当前 firmware 位于 `firmware/zapper_v5/zapper_v5.ino`，说明位于 `firmware/zapper_v5/README.md`。写入 firmware 后：

1. 打开 **设备** 视图。
2. 选择 COM 端口并点击 **连接**。
3. 等待状态变为 **就绪**。
4. 发送今天的会话，或在手动模式下启动单个值。
5. 检查开发板上的连接，然后按下实体按钮；只有此时输出才会开始。

所选端口会记录在本地 `data/device.json` 文件中。配置会话会保存独立、精确的 `device_steps`；例如“30 kHz”仍然是便于人阅读的文本，而开发板实际接收 `30000000` 毫赫兹以及以毫秒表示的时长。

### LCD firmware 语言

Firmware 5.1.0 具有 30 个独立语言变体，它们都由同一套代码生成。每个 Arduino sketch 只包含一套 LCD 文本。使用拉丁字母的语言拥有各自的短文本，并以安全的 ASCII 保存。对于西里尔字母以及其他普通 LCD1602/HD44780 无法可靠显示的文字系统，对应变体使用英文 LCD 界面。完整列表位于 `firmware/LANGUAGES.md`。

命令 `go run ./tools/firmware_i18n` 会在 `build/generated/firmware/` 中生成全部 sketch。正常的 `build.ps1` 流程会自动完成这一步，并把这些变体加入 portable 版本。

### 从应用写入 firmware

**设备 → Firmware** 区域显示检测到的版本、最新版本、firmware 变体语言以及 LCD 语言。用户选择 Arduino Nano 的新 bootloader 或旧 bootloader，然后明确点击 **写入 firmware**；应用在启动时绝不会自动向开发板写入 firmware。

编译和上传由 `arduino-cli` 完成。Zapper 会在 `tools/arduino-cli/`、EXE 文件旁、`PATH` 以及常见 Arduino IDE 目录中查找它。如果工具不存在，应用会明确提示，并保持写入按钮不可用。编译还需要所使用的 `arduino-cli` 安装中提供 `arduino:avr` core 和 `LiquidCrystal_I2C` 库。

### 语言检测与 firmware 选择

启动时，应用读取 WebView2/Windows 环境语言（`navigator.languages`），并映射到 30 个受支持代码之一。如果系统语言不受支持，则选择英语。在 **自动（Windows）** 模式下，每次启动都会检查语言；手动选择会保存在 `data/settings.json` 中，直到重新启用自动模式。

同一个语言代码也是 firmware 写入界面的默认选择。对于 LCD1602 不支持的语言，应用仍会选择以用户语言标记的变体，但会提示 LCD 文本将使用英语。应用启动时绝不会自动写入 firmware；必须由用户明确点击写入，以避免意外覆盖 Arduino 上已经存在的其他程序。

## 构建

需要安装 Go。最简单的方法是在项目根目录中运行：

```text
build.bat
```

也可以在 PowerShell 中运行：

```powershell
.\build.ps1
```

脚本会运行测试和代码分析，构建 `build/generated/Zapper-dev.exe`，并准备不带控制台窗口的 portable `build/Zapper/Zapper.exe`。

## 项目结构

- `app/` — Go 代码、HTML/CSS/JS 界面、指南和频率数据库。
- `firmware/zapper_v5/` — 当前 Arduino firmware。
- `data/` — 活动配置、进度、归档、设备设置和自动备份。
- `locales/` — 纳入版本控制的界面和指南翻译，供开发使用并复制到发布版本中。
- `build/Zapper/` — 可直接复制到另一台电脑的 portable 版本。