# dango

dango は Discord 用の小さなユーティリティ bot です。

- `/random` でメンションしたユーザーやロールから抽選できます。
- `/remind` で Todoist 風の自然文リマインダーを登録できます。
- リマインダーは SQLite に保存され、bot 再起動後も維持されます。

## 起動

```sh
cp .env.example .env
```

`.env` の値を設定します。

```env
DISCORD_TOKEN=your_discord_bot_token
DANGO_TIMEZONE=Asia/Tokyo
DANGO_DATABASE_PATH=dango.sqlite3
```

環境変数を読み込んで起動します。

```sh
go run ./cmd/dango
```

## コマンド

```text
/random mentions:"@A @B @C"
/random mentions:"@A @B @C /2"
/random mentions:"@everyone"
/random mentions:"@role /3"
```

```text
/remind text:"明日9時に請求書確認" target:me
/remind text:"１０分後に水を飲む" target:me
/remind text:"毎週 月曜 10:30 に朝会" target:channel
/remind text:"every monday 10am standup" target:channel
/reminders
/remind-cancel id:1
```

## ドキュメント

- [アーキテクチャ](docs/architecture.md)
- [設定](docs/configuration.md)
- [リマインダー](docs/reminders.md)
