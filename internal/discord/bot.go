package discord

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/yashikota/dango/internal/config"
	"github.com/yashikota/dango/internal/random"
	"github.com/yashikota/dango/internal/reminder"
)

type Bot struct {
	session   *discordgo.Session
	config    config.Config
	reminders *reminder.Service
}

func NewBot(cfg config.Config, reminders *reminder.Service) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("create Discord session: %w", err)
	}

	bot := &Bot{
		session:   session,
		config:    cfg,
		reminders: reminders,
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentGuildMembers
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onInteraction)
	session.AddHandler(bot.onMessage)
	return bot, nil
}

func (b *Bot) Open() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open Discord connection: %w", err)
	}
	return b.registerCommands()
}

func (b *Bot) Close() error {
	return b.session.Close()
}

func (b *Bot) onReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s#%s", r.User.Username, r.User.Discriminator)
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "random",
			Description: "メンションしたユーザーからランダムに抽選します",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mentions",
					Description: "@user @role @everyone /count",
					Required:    true,
				},
			},
		},
		{
			Name:        "remind",
			Description: "Todoist風の自然文でリマインダーを登録します",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "例: 明日9時に請求書確認 / every monday 10am standup",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "target",
					Description: "通知先",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "自分宛", Value: string(reminder.TargetMe)},
						{Name: "このチャンネル宛", Value: string(reminder.TargetChannel)},
					},
				},
			},
		},
		{
			Name:        "reminders",
			Description: "自分が登録した有効なリマインダーを表示します",
		},
		{
			Name:        "remind-cancel",
			Description: "自分が登録したリマインダーをキャンセルします",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "キャンセルするリマインダーID",
					Required:    true,
				},
			},
		},
	}

	for _, cmd := range commands {
		if _, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd); err != nil {
			return fmt.Errorf("register /%s: %w", cmd.Name, err)
		}
		log.Printf("Registered command: /%s", cmd.Name)
	}
	return nil
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "random":
		b.handleRandom(s, i)
	case "remind":
		b.handleRemind(s, i)
	case "reminders":
		b.handleReminders(s, i)
	case "remind-cancel":
		b.handleRemindCancel(s, i)
	}
}

func (b *Bot) handleRandom(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.deferResponse(s, i, false)

	if i.GuildID == "" {
		b.followup(s, i, "⚠️ 抽選機能はサーバー内で実行してください", false)
		return
	}

	input := stringOption(i, "mentions")
	content, err := random.Handle(s, i.GuildID, s.State.User.ID, input)
	if err != nil {
		b.followup(s, i, "⚠️ "+err.Error(), false)
		return
	}
	b.followup(s, i, content, false)
}

func (b *Bot) handleRemind(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.deferResponse(s, i, true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := interactionUserID(i)
	if userID == "" || i.GuildID == "" {
		b.followup(s, i, "⚠️ リマインダーはサーバー内で実行してください", true)
		return
	}

	rem, err := b.reminders.Create(ctx, reminder.CreateRequest{
		GuildID:       i.GuildID,
		ChannelID:     i.ChannelID,
		CreatorUserID: userID,
		Target:        reminder.Target(stringOption(i, "target")),
		Text:          stringOption(i, "text"),
	})
	if err != nil {
		b.followup(s, i, "⚠️ "+err.Error(), true)
		return
	}

	b.followup(s, i, b.reminders.FormatCreated(rem), true)
}

func (b *Bot) handleReminders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.deferResponse(s, i, true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := interactionUserID(i)
	if userID == "" || i.GuildID == "" {
		b.followup(s, i, "⚠️ リマインダーはサーバー内で実行してください", true)
		return
	}

	items, err := b.reminders.ListActiveByCreator(ctx, i.GuildID, userID)
	if err != nil {
		b.followup(s, i, "⚠️ リマインダー一覧の取得に失敗しました", true)
		log.Printf("list reminders failed: %v", err)
		return
	}
	b.followup(s, i, b.reminders.FormatList(items), true)
}

func (b *Bot) handleRemindCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.deferResponse(s, i, true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := interactionUserID(i)
	if userID == "" || i.GuildID == "" {
		b.followup(s, i, "⚠️ リマインダーはサーバー内で実行してください", true)
		return
	}

	id := intOption(i, "id")
	ok, err := b.reminders.CancelByCreator(ctx, id, i.GuildID, userID)
	if err != nil {
		b.followup(s, i, "⚠️ リマインダーのキャンセルに失敗しました", true)
		log.Printf("cancel reminder failed: %v", err)
		return
	}
	if !ok {
		b.followup(s, i, "指定された有効なリマインダーが見つかりません", true)
		return
	}
	b.followup(s, i, fmt.Sprintf("リマインダー `%d` をキャンセルしました", id), true)
}

func (b *Bot) SendReminder(ctx context.Context, rem reminder.Reminder) error {
	content := reminder.FormatReminderMessage(rem)
	withCtx := discordgo.WithContext(ctx)
	switch rem.Target {
	case reminder.TargetMe:
		channel, err := b.session.UserChannelCreate(rem.CreatorUserID, withCtx)
		if err == nil {
			if _, sendErr := b.session.ChannelMessageSend(channel.ID, content, withCtx); sendErr == nil {
				return nil
			} else {
				err = sendErr
			}
		}

		fallback := fmt.Sprintf("<@%s>\n%s", rem.CreatorUserID, content)
		if _, fallbackErr := b.session.ChannelMessageSend(rem.ChannelID, fallback, withCtx); fallbackErr != nil {
			return fmt.Errorf("send DM failed: %v; fallback failed: %w", err, fallbackErr)
		}
		return nil
	case reminder.TargetChannel:
		_, err := b.session.ChannelMessageSend(rem.ChannelID, content, withCtx)
		return err
	default:
		return fmt.Errorf("unknown reminder target %q", rem.Target)
	}
}

func (b *Bot) deferResponse(s *discordgo.Session, i *discordgo.InteractionCreate, ephemeral bool) {
	data := &discordgo.InteractionResponseData{}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: data,
	}); err != nil {
		log.Printf("Error deferring interaction: %v", err)
	}
}

func (b *Bot) followup(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	params := &discordgo.WebhookParams{Content: content}
	if ephemeral {
		params.Flags = discordgo.MessageFlagsEphemeral
	}
	if _, err := s.FollowupMessageCreate(i.Interaction, true, params); err != nil {
		log.Printf("Error sending followup: %v", err)
	}
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	botMentioned := false
	for _, u := range m.Mentions {
		if u.ID == s.State.User.ID {
			botMentioned = true
			break
		}
	}
	if !botMentioned {
		return
	}

	if _, err := s.ChannelMessageSend(m.ChannelID, b.infoMessage()); err != nil {
		log.Printf("send info message failed: %v", err)
	}
}

func (b *Bot) infoMessage() string {
	return "🍡 **dango の使い方** " + fmt.Sprintf("v%s", b.config.Version) + "\n" +
		"`/random @A @B @C` → 1人抽選\n" +
		"`/random @A @B @C /2` → 2人抽選\n" +
		"`/random @everyone` → 全員から抽選\n" +
		"`/random @ロール /3` → ロールから3人抽選\n" +
		"`/remind text:\"明日9時に請求書確認\" target:me` → 自分宛リマインダー\n" +
		"`/remind text:\"every monday 10am standup\" target:channel` → 繰り返しチャンネル通知\n" +
		"`/reminders` → 有効なリマインダー一覧\n" +
		"`/remind-cancel id:1` → リマインダーをキャンセル\n"
}

func stringOption(i *discordgo.InteractionCreate, name string) string {
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == name {
			return opt.StringValue()
		}
	}
	return ""
}

func intOption(i *discordgo.InteractionCreate, name string) int64 {
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == name {
			return opt.IntValue()
		}
	}
	return 0
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}
