package reminder

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
	loc  *time.Location
	now  func() time.Time
}

type CreateRequest struct {
	GuildID       string
	ChannelID     string
	CreatorUserID string
	Target        Target
	Text          string
}

func NewService(repo *Repository, loc *time.Location) *Service {
	return &Service{
		repo: repo,
		loc:  loc,
		now:  time.Now,
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Reminder, error) {
	if req.Target != TargetMe && req.Target != TargetChannel {
		return Reminder{}, fmt.Errorf("target は me または channel を指定してください")
	}
	parsed, err := Parse(req.Text, s.now(), s.loc)
	if err != nil {
		return Reminder{}, err
	}

	now := s.now().UTC()
	rem := Reminder{
		GuildID:       req.GuildID,
		ChannelID:     req.ChannelID,
		CreatorUserID: req.CreatorUserID,
		Target:        req.Target,
		Message:       parsed.Message,
		NextRunAt:     parsed.NextRunAt.UTC(),
		Recurrence:    parsed.Recurrence,
		Status:        StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	id, err := s.repo.Create(ctx, rem)
	if err != nil {
		return Reminder{}, err
	}
	rem.ID = id
	return rem, nil
}

func (s *Service) ListActiveByCreator(ctx context.Context, guildID, userID string) ([]Reminder, error) {
	return s.repo.ListActiveByCreator(ctx, guildID, userID)
}

func (s *Service) CancelByCreator(ctx context.Context, id int64, guildID, userID string) (bool, error) {
	return s.repo.CancelByCreator(ctx, id, guildID, userID)
}

func (s *Service) FormatTime(t time.Time) string {
	return t.In(s.loc).Format("2006-01-02 15:04")
}

func (s *Service) FormatCreated(rem Reminder) string {
	target := "あなた宛"
	if rem.Target == TargetChannel {
		target = "このチャンネル宛"
	}
	repeat := ""
	if rem.Recurrence != nil {
		repeat = "\n繰り返し: " + describeRecurrence(*rem.Recurrence)
	}
	return fmt.Sprintf("🍡 リマインダーを登録しました\nID: `%d`\n宛先: %s\n日時: %s\n内容: %s%s",
		rem.ID, target, s.FormatTime(rem.NextRunAt), rem.Message, repeat)
}

func (s *Service) FormatList(reminders []Reminder) string {
	if len(reminders) == 0 {
		return "有効なリマインダーはありません"
	}

	var b strings.Builder
	b.WriteString("🍡 **有効なリマインダー**\n")
	for _, rem := range reminders {
		target := "me"
		if rem.Target == TargetChannel {
			target = "channel"
		}
		b.WriteString(fmt.Sprintf("`%d` [%s] %s - %s", rem.ID, target, s.FormatTime(rem.NextRunAt), rem.Message))
		if rem.Recurrence != nil {
			b.WriteString(" (" + describeRecurrence(*rem.Recurrence) + ")")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func describeRecurrence(r Recurrence) string {
	timeText := fmt.Sprintf("%02d:%02d", r.Hour, r.Minute)
	switch r.Kind {
	case RecurrenceDaily:
		return "毎日 " + timeText
	case RecurrenceWeekday:
		return "平日 " + timeText
	case RecurrenceWeekly:
		if r.Weekday == nil {
			return "毎週 " + timeText
		}
		return fmt.Sprintf("毎週%s %s", japaneseWeekdayName(time.Weekday(*r.Weekday)), timeText)
	default:
		return string(r.Kind)
	}
}

func japaneseWeekdayName(w time.Weekday) string {
	switch w {
	case time.Sunday:
		return "日曜"
	case time.Monday:
		return "月曜"
	case time.Tuesday:
		return "火曜"
	case time.Wednesday:
		return "水曜"
	case time.Thursday:
		return "木曜"
	case time.Friday:
		return "金曜"
	case time.Saturday:
		return "土曜"
	default:
		return "曜日不明"
	}
}
