# worklog

Mac で「何に時間を使ったか」を自動記録する Go 製 CLI / デーモン。
フォアグラウンドアプリとウィンドウタイトルを定期的に観測し、ローカルの SQLite に記録します。手動入力ゼロで「今日 / 今週 / 今月、どのプロジェクトに何時間使ったか」を可視化できます。SwiftBar 連携でメニューバーに当日サマリを常駐表示できます。

> **プライバシー第一** — データ送信は一切ありません。すべてローカル (`~/.local/share/worklog`) に保存され、`worklog wipe` でいつでも削除できます。キーログもスクリーンショットも取りません。

## 特徴

- 🕒 **自動記録** — フォアグラウンドアプリ + ウィンドウタイトルを 5 秒ごとに記録
- 📊 **集計** — アプリ別 / プロジェクト別の作業時間、集中ブロック検出
- 🏷️ **プロジェクト推論** — TOML の正規表現ルールでウィンドウタイトルからプロジェクトを自動分類
- 💤 **アイドル検出** — `ioreg` の `HIDIdleTime` で離席を自動判定
- 📌 **SwiftBar 連携** — メニューバーに `⏱ 4h 23m` を常時表示
- 🔒 **プライバシー設計** — タイトル記録のオプトアウト、機微なアプリの除外
- 📦 **cgo 不要の単一バイナリ** — Apple Silicon / Intel 両対応

## 動作環境

macOS 専用です (`osascript` / `ioreg` / `launchd` に依存)。Linux / Windows は対応外です。

## インストール

```bash
# ソースからビルド
git clone https://github.com/reisuta/worklog.git
cd worklog
make install        # $GOPATH/bin に worklog / worklog-statusbar を配置
```

初回実行時に **アクセシビリティ** 権限を求められます。
システム設定 > プライバシーとセキュリティ > アクセシビリティ で、worklog (またはそれを起動するターミナル) を許可してください。

## クイックスタート

```bash
worklog start        # デーモンをバックグラウンド起動
worklog today        # 当日サマリを表示
worklog stop         # 停止
```

出力例:

```
今日 (2026-05-29)
合計 4h 23m (集中 2h 15m、離席 1h 5m)

プロジェクト
  blog    2h 30m  ████████░░░░
  my-app  1h 10m  ████░░░░░░░░
  sns        25m  █░░░░░░░░░░░

集中ブロック
  10:15-11:23 (1h 8m) blog Cursor
  14:30-15:42 (1h 12m) my-app Cursor
```

## コマンド一覧

| コマンド | 説明 |
|---|---|
| `worklog start` / `stop` / `status` | デーモンの起動 / 停止 / 状態確認 |
| `worklog daemon` | フォアグラウンド実行 (launchd / 内部用) |
| `worklog today` / `yesterday` / `week` / `month` | 期間別サマリ |
| `worklog report --from 2026-05-01 --to 2026-05-29` | 期間指定レポート |
| `worklog statusbar` | SwiftBar 形式で当日サマリを出力 |
| `worklog export --format csv --month 2026-05` | CSV / JSON エクスポート |
| `worklog wipe --before 2026-01-01` / `--all` | データ削除 |
| `worklog rule list` / `rule edit` | ルールの確認 / 編集 |
| `worklog config` | 設定ファイルを `$EDITOR` で開く |
| `worklog install-launchd` / `uninstall-launchd` | ログイン時の自動起動の登録 / 解除 |

## 自動起動 (launchd)

```bash
worklog install-launchd     # ~/Library/LaunchAgents に登録し、即時ロード
worklog uninstall-launchd   # 登録解除
```

## SwiftBar 連携

```bash
brew install --cask swiftbar
mkdir -p ~/SwiftBar
ln -s "$(which worklog-statusbar)" ~/SwiftBar/worklog.5s.bin   # 5 秒間隔で更新
```

メニューバーに `⏱ 4h 23m` のように表示され、クリックするとプロジェクト別の内訳と操作メニューが開きます。集中ブロック中は `🔥 47m (blog)` のように表示が切り替わります。

## 設定

`~/.config/worklog/config.toml` (`worklog config` で作成・編集):

```toml
[tracker]
poll_interval_seconds = 5       # 何秒ごとに記録するか
idle_threshold_seconds = 300    # 5 分動かなければ離席
focus_block_min_minutes = 15    # 連続作業 15 分以上を集中ブロック

[storage]
db_path = "~/.local/share/worklog/worklog.db"

[statusbar]
format = "⏱ {total}"
focus_format = "🔥 {duration} ({project})"

[privacy]
record_window_title = true      # false にするとアプリ名のみ記録
exclude_apps = ["1Password"]    # タイトルを記録しないアプリ
```

## ルール (プロジェクト推論)

`~/.config/worklog/rules.toml` (`worklog rule edit` で作成・編集):

```toml
[[rules]]
match_app = "Cursor"                  # アプリ名の完全一致 (空なら任意)
match_title = ".*/my-blog.*"          # タイトルの正規表現 (空なら任意)
project = "blog"
category = "work"

[[rules]]
match_app = "Google Chrome"
match_title = "(?i).*(twitter|x\\.com).*"
project = "sns"
category = "sns"

[default]
project = "other"
category = "other"
```

ルールは上から順に評価され、最初にマッチしたものが採用されます。

## プライバシー

| 項目 | 方針 |
|---|---|
| データ送信 | **一切なし** (オフラインで完結) |
| 記録内容 | アプリ名・ウィンドウタイトル・時刻のみ |
| キーログ / スクリーンショット | 取らない |
| ストレージ | ローカル SQLite (`~/.local/share/worklog/worklog.db`) |
| 削除 | `worklog wipe` でいつでも削除可 |
| タイトル記録 | `record_window_title = false` でオフ、`exclude_apps` で個別除外 |

## 開発

```bash
make build    # bin/ にビルド
make test     # go test -race ./...
make vet
make lint     # golangci-lint (要インストール)
```

技術スタック: Go / `modernc.org/sqlite` (cgo 不要) / `pelletier/go-toml`。
