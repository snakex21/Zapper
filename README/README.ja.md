**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

新しいバージョンのアプリケーションは1つのウィンドウで動作し、Python、Node.js、Wailsを必要としません。ボードを接続せずにプランナー兼ログとして使うことも、USB経由でArduino Nanoを制御することもできます。

## ライセンスと責任

コード、firmware、配線図、ドキュメントは、**PolyForm Noncommercial 1.0.0** ライセンスの下で非商用利用向けに公開されています。このライセンスが許可する範囲で使用、調査、変更、配布できますが、作者の別途許可なしに本プロジェクトを商用利用することはできません。詳細は `LICENSE` ファイルを参照してください。

本プロジェクトは、個人の実験やDIY用途向けに無保証で提供されます。正しい組み立て、改造、機器の使用方法については利用者が責任を負います。作者は、ハードウェアの損傷、その他の損害、誤った組み立てや使用による結果について責任を負わず、特定の健康上の効果を保証しません。

## アプリケーションの起動

portable版のフォルダーから `Zapper.exe` を実行します。固定の人物とその識別子は `data/persons.json`、有効なプロファイルは `data/profiles.json` に保存され、各実行には `data/progress/` 内に個別ファイルがあります。完了した実行は、`profile.json` と `progress.json` を含む `data/archive/<id>/` フォルダーへ移動されます。ボード設定は `data/device.json`、検出または選択された言語を含むアプリ設定は `data/settings.json` に保存されます。バックアップはローカルの `backups/` サブフォルダーに残ります。すべてEXEの隣に保存され、AppData、Documents、Windowsレジストリには書き込みません。

**プロファイル** 画面では、人物の追加、クリップボードへコピーできるAI用コンテキスト文章の生成、AIモデルが返した簡略化JSONの貼り付けができます。この形式では周波数を `frequency_hz` で指定します。アプリはプロファイルを検証し、プレビューを表示し、確認後にのみ新しい `run_id` を作成します。その人物の以前の有効な実行は先にアーカイブされます。

プロファイルセッション中、**一時停止** ボタンは現在のステップの残りと、その後のすべてのステップをローカル進捗に保存します。再開すると、変更していないfirmwareへ短縮されたシーケンスを送り、ボード上で再び物理的な確認を求めます。**停止** は部分的な進捗を取り消し、完全なセッションを再実行できる状態に戻します。

スキップしたセッションは期限超過としてキューに残ります。プログラム規則では、パート数、シリーズ内の休止、完全なセッション同士の間隔、セッション後のクールダウン、同日の他プログラムとの互換性を定義します。期限超過セッションがないプロファイルは計画完了後に自動でアーカイブされ、**プログラムを終了** を使えばそれより早く閉じることもできます。

## アプリケーションの言語

起動時、アプリはWindows/WebView2の言語を読み取り、対応する30言語のいずれかに割り当てます。設定が **自動（Windows）** の間は、起動するたびに言語を検出します。左側パネルで手動選択した言語は `data/settings.json` に保存され、自動モードを再び選ぶまで自動変更は行われません。

アプリケーションの言語はfirmwareバリアントの既定言語にもなります。標準LCD1602/HD44780で安定して表示できない文字体系の場合、アプリは対応するfirmwareバリアントを選択し、LCDの表示だけ英語を使います。デスクトップ側のインターフェースは選択された言語のままです。

## ArduinoとUSB

現在のfirmwareは `firmware/zapper_v5/zapper_v5.ino` にあり、説明は `firmware/zapper_v5/README.md` にあります。firmwareを書き込んだ後は次の手順です。

1. **デバイス** 画面を開きます。
2. COMポートを選び、**接続** をクリックします。
3. **準備完了** 状態になるまで待ちます。
4. 今日のセッションを送信するか、手動モードで単一の値を開始します。
5. ボード上の接続を確認してから物理ボタンを押します。出力はその後に開始します。

選択したポートはローカルの `data/device.json` に記憶されます。プロファイルセッションは個別かつ正確な `device_steps` を保持します。「30 kHz」のような説明は人間向けのテキストとして残り、ボードには `30000000` ミリヘルツとミリ秒単位の時間が送られます。

### LCD firmwareの言語

Firmware 5.1.0には、1つのコードベースから作られる30個の独立した言語バリアントがあります。各Arduino sketchにはLCDテキストが1言語分だけ入ります。ラテン文字を使う言語には、安全なASCIIとして保存された短い専用文言があります。キリル文字など、一般的なLCD1602/HD44780で安定して表示できない文字体系では、対応するバリアントが英語のLCDインターフェースを使います。完全な一覧は `firmware/LANGUAGES.md` にあります。

`go run ./tools/firmware_i18n` コマンドはすべてのsketchを `build/generated/firmware/` に作成します。通常の `build.ps1` 処理ではこれを自動実行し、portable版に各バリアントを含めます。

### アプリからfirmwareを書き込む

**デバイス → Firmware** セクションには、検出されたバージョン、最新バージョン、firmwareバリアントの言語、LCD言語が表示されます。利用者はArduino Nanoの新しいbootloaderまたは古いbootloaderを選び、明示的に **Firmwareを書き込む** をクリックします。アプリ起動時に自動でボードへfirmwareを書き込むことはありません。

コンパイルとアップロードには `arduino-cli` を使います。Zapperは `tools/arduino-cli/`、EXEの隣、`PATH`、一般的なArduino IDEの場所を検索します。ツールが見つからない場合はアプリが明確に表示し、書き込みボタンは無効のままです。コンパイルには、使用する `arduino-cli` 環境に `arduino:avr` coreと `LiquidCrystal_I2C` ライブラリも必要です。

### 言語検出とfirmwareの選択

起動時、アプリはWebView2/Windows環境の言語（`navigator.languages`）を読み取り、30個の対応コードのいずれかに割り当てます。システム言語が未対応なら英語を選択します。**自動（Windows）** モードでは起動のたびに言語を確認し、手動選択は自動モードを再有効化するまで `data/settings.json` に保存されます。

同じ言語コードがfirmware書き込み画面の既定選択になります。LCD1602で表示できない言語でも、アプリは利用者の言語名が付いたバリアントを選択し、LCD表示が英語になることを知らせます。アプリ起動時にfirmwareが自動で書き込まれることはありません。Arduinoにすでに入っている別のプログラムを誤って上書きしないよう、書き込みには利用者による明示的なクリックが必要です。

## ビルド

Goが必要です。最も簡単なのは、プロジェクトのルートフォルダーで次を実行する方法です。

```text
build.bat
```

PowerShellでは次を実行できます。

```powershell
.\build.ps1
```

スクリプトはテストとコード解析を実行し、`build/generated/Zapper-dev.exe` をビルドして、コンソールウィンドウなしのportable `build/Zapper/Zapper.exe` を準備します。

## プロジェクト構成

- `app/` — Goコード、HTML/CSS/JSインターフェース、ガイド、周波数データベース。
- `firmware/zapper_v5/` — 現在のArduino firmware。
- `data/` — 有効なプロファイル、進捗、アーカイブ、デバイス設定、自動バックアップ。
- `locales/` — バージョン管理されたUIとガイドの翻訳。開発時に使用し、リリースへコピーします。
- `build/Zapper/` — 別のコンピューターへコピーできる完成済みportable版。