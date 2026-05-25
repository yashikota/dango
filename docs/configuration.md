# 設定

dango は環境変数で設定します。

## 環境変数

| 名前 | 必須 | デフォルト | 説明 |
| --- | --- | --- | --- |
| `DISCORD_TOKEN` | はい | なし | Discord bot token |
| `DANGO_TIMEZONE` | いいえ | `Asia/Tokyo` | 自然文の絶対日時を解釈するタイムゾーン |
| `DANGO_DATABASE_PATH` | いいえ | `dango.sqlite3` | SQLite database の保存先 |

## Discord 側の設定

Discord Developer Portal で bot を作成し、token を `DISCORD_TOKEN` に設定します。

必要な intent:

- Server Members Intent
- Message Content Intent

bot には、slash command の登録、メッセージ送信、DM送信に必要な権限を付けてください。
DM送信に失敗した場合、個人宛リマインダーは登録元チャンネルでユーザーにメンションして通知します。

## SQLite

`DANGO_DATABASE_PATH` の場所に SQLite ファイルを作成します。
初回起動時に `reminders` テーブルと index を自動作成します。

開発環境では次のように起動できます。

```sh
DISCORD_TOKEN=... DANGO_TIMEZONE=Asia/Tokyo DANGO_DATABASE_PATH=dango.sqlite3 go run ./cmd/dango
```
