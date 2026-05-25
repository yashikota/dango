# リマインダー

dango のリマインダーは、Todoist 風に本文へ日時を混ぜて入力します。

## 基本コマンド

```text
/remind text:"明日9時に請求書確認" target:me
/remind text:"毎週 月曜 10:30 に朝会" target:channel
/reminders
/remind-cancel id:1
```

`target` は次のどちらかです。

| 値 | 通知先 |
| --- | --- |
| `me` | 登録者へのDM。DMに失敗した場合は登録元チャンネルでメンション |
| `channel` | 登録したチャンネル |

## 対応構文

### 日本語

```text
10分後に水を飲む
2時間後にレビュー
明日9時に請求書確認
来週月曜 10:00 に定例
2026-05-26 09:00 に提出
2026年5月26日 9時30分 に提出
毎日9時に日報
毎週月曜10時に朝会
平日9時にスタンドアップ
```

全角数字、全角コロン、全角スペースも扱えます。

```text
１０分後に水を飲む
２０２６／０５／２６　０９：３０　に提出
毎週　月曜　１０：３０　に朝会
```

### 英語

```text
in 10m drink water
in 2 hours review
tomorrow 9am submit report
next monday 10:00 meeting
2026-05-26 09:00 submit
every day 9am daily report
every monday 10am standup
every weekday 9am standup
```

## 解釈ルール

- 日時として解釈した部分を取り除き、残りをリマインダー本文として保存します。
- 時刻を省略した場合は `09:00` として扱います。
- 絶対日時は `DANGO_TIMEZONE` のタイムゾーンで解釈します。
- 過去日時や本文が空になる入力は登録しません。
- Todoist 完全互換ではなく、よく使う構文を段階的に増やす方針です。
