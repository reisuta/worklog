package cli

import "path/filepath"

func dirOf(path string) string { return filepath.Dir(path) }

const sampleConfig = `# worklog 設定ファイル
# ~/.config/worklog/config.toml

[tracker]
poll_interval_seconds = 5      # 何秒ごとに記録するか
idle_threshold_seconds = 300   # 5分動かなければ離席とみなす
focus_block_min_minutes = 15   # 連続作業 15分以上を集中ブロックとする

[storage]
db_path = "~/.local/share/worklog/worklog.db"

[reports]
auto_daily_markdown = false
report_dir = "~/Documents/worklog"

[statusbar]
format = "⏱ {total}"               # {total} が当日合計に置換される
focus_format = "🔥 {duration} ({project})"

[privacy]
record_window_title = true         # false にするとアプリ名のみ記録
exclude_apps = ["1Password"]       # タイトルを記録しないアプリ
`

const sampleRules = `# worklog ルール定義
# ~/.config/worklog/rules.toml
#
# match_app   : アプリ名の完全一致 (空なら任意)
# match_title : ウィンドウタイトルの正規表現 (空なら任意)
# 上から順に評価し、最初にマッチしたルールを採用します。

[[rules]]
match_app = "Cursor"
match_title = ".*/my-blog.*"
project = "blog"
category = "work"

[[rules]]
match_app = "Google Chrome"
match_title = ".*(twitter|x\\.com|facebook|instagram).*"
project = "sns"
category = "sns"

[[rules]]
match_app = "Slack"
project = "communication"
category = "work"

# どのルールにもマッチしない場合
[default]
project = "other"
category = "other"
`
