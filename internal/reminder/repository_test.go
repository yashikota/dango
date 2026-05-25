package reminder

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryCreateListDueCancel(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)

	id, err := repo.Create(ctx, Reminder{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		CreatorUserID: "user-1",
		Target:        TargetMe,
		Message:       "請求書確認",
		NextRunAt:     now.Add(time.Minute),
		Status:        StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id == 0 {
		t.Fatal("Create() returned zero id")
	}

	items, err := repo.ListActiveByCreator(ctx, "guild-1", "user-1")
	if err != nil {
		t.Fatalf("ListActiveByCreator() error = %v", err)
	}
	if len(items) != 1 || items[0].Message != "請求書確認" {
		t.Fatalf("ListActiveByCreator() = %#v", items)
	}

	due, err := repo.Due(ctx, now, 10)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("Due() before scheduled time = %d items, want 0", len(due))
	}

	due, err = repo.Due(ctx, now.Add(2*time.Minute), 10)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if len(due) != 1 || due[0].ID != id {
		t.Fatalf("Due() = %#v, want reminder %d", due, id)
	}

	ok, err := repo.CancelByCreator(ctx, id, "guild-1", "user-1")
	if err != nil {
		t.Fatalf("CancelByCreator() error = %v", err)
	}
	if !ok {
		t.Fatal("CancelByCreator() = false, want true")
	}

	items, err = repo.ListActiveByCreator(ctx, "guild-1", "user-1")
	if err != nil {
		t.Fatalf("ListActiveByCreator() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("ListActiveByCreator() after cancel = %d items, want 0", len(items))
	}
}

func TestRepositoryReschedule(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)

	id, err := repo.Create(ctx, Reminder{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		CreatorUserID: "user-1",
		Target:        TargetChannel,
		Message:       "朝会",
		NextRunAt:     now,
		Status:        StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	next := now.Add(24 * time.Hour)
	if err := repo.Reschedule(ctx, id, next); err != nil {
		t.Fatalf("Reschedule() error = %v", err)
	}

	due, err := repo.Due(ctx, now.Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("Due() after reschedule = %d items, want 0", len(due))
	}

	due, err = repo.Due(ctx, next.Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if len(due) != 1 || !due[0].NextRunAt.Equal(next) {
		t.Fatalf("Due() = %#v, want next run at %s", due, next)
	}
}

func openTestRepository(t *testing.T) *Repository {
	t.Helper()
	repo, err := OpenRepository(filepath.Join(t.TempDir(), "dango-test.sqlite3"))
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return repo
}
