package reminder

import (
	"context"
	"testing"
	"time"
)

type fakeSender struct {
	sent []Reminder
	err  error
}

func (f *fakeSender) SendReminder(_ context.Context, rem Reminder) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, rem)
	return nil
}

func TestSchedulerCompletesOneShotReminder(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)

	if _, err := repo.Create(ctx, Reminder{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		CreatorUserID: "user-1",
		Target:        TargetMe,
		Message:       "水を飲む",
		NextRunAt:     now.Add(-time.Minute),
		Status:        StatusActive,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sender := &fakeSender{}
	scheduler := NewScheduler(repo, sender)
	scheduler.now = func() time.Time { return now }

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.sent))
	}

	due, err := repo.Due(ctx, now, 10)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due after Tick() = %d, want 0", len(due))
	}
}

func TestSchedulerReschedulesRecurringReminder(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	loc := mustLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, loc)

	if _, err := repo.Create(ctx, Reminder{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		CreatorUserID: "user-1",
		Target:        TargetChannel,
		Message:       "朝会",
		NextRunAt:     now.Add(-time.Minute),
		Recurrence: &Recurrence{
			Kind:     RecurrenceDaily,
			Hour:     9,
			Minute:   0,
			Timezone: loc.String(),
		},
		Status: StatusActive,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sender := &fakeSender{}
	scheduler := NewScheduler(repo, sender)
	scheduler.now = func() time.Time { return now }

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.sent))
	}

	items, err := repo.ListActiveByCreator(ctx, "guild-1", "user-1")
	if err != nil {
		t.Fatalf("ListActiveByCreator() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("active reminders = %d, want 1", len(items))
	}
	want := time.Date(2026, 5, 25, 9, 0, 0, 0, loc).UTC()
	if !items[0].NextRunAt.Equal(want) {
		t.Fatalf("NextRunAt = %s, want %s", items[0].NextRunAt, want)
	}
}
