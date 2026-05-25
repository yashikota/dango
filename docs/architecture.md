# アーキテクチャ

dango は Discord bot の入出力、機能ロジック、永続化を分離した Go アプリケーションです。

## ディレクトリ構成

```text
cmd/dango/              起動エントリポイント
internal/config/        環境変数とタイムゾーン設定
internal/discord/       Discord セッション、slash command、handler
internal/random/        抽選機能
internal/reminder/      リマインダーのパース、DB、scheduler
docs/                   日本語ドキュメント
```

## 起動フロー

1. `cmd/dango` が `DISCORD_TOKEN` などの設定を読み込みます。
2. SQLite repository を開き、必要なテーブルを作成します。
3. Discord bot を起動し、slash command を登録します。
4. reminder scheduler が一定間隔で期限到来リマインダーを確認します。

## 責務

`internal/discord` は Discord API との接続とメッセージ整形だけを担当します。
機能追加時は、コマンド入力を各機能パッケージへ渡し、結果だけを Discord に返す形を保ちます。

`internal/reminder` は Discord に依存しない設計です。
送信処理は `Sender` interface 経由にしているため、scheduler のテストでは fake sender を使えます。

## データフロー

`/remind` が実行されると、入力テキストを parser が解釈し、本文・次回実行時刻・繰り返し条件に分けます。
repository が SQLite に保存し、scheduler が期限到来後に sender へ渡します。
単発リマインダーは送信後に `completed`、繰り返しリマインダーは次回実行時刻へ更新されます。
