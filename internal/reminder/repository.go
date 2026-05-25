package reminder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
}

func OpenRepository(path string) (*Repository, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	repo := &Repository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS reminders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	guild_id TEXT NOT NULL,
	channel_id TEXT NOT NULL,
	creator_user_id TEXT NOT NULL,
	target TEXT NOT NULL,
	message TEXT NOT NULL,
	next_run_at TEXT NOT NULL,
	recurrence TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reminders_due ON reminders(status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_reminders_owner ON reminders(guild_id, creator_user_id, status);
`)
	return err
}

func (r *Repository) Create(ctx context.Context, rem Reminder) (int64, error) {
	now := time.Now().UTC()
	if rem.CreatedAt.IsZero() {
		rem.CreatedAt = now
	}
	if rem.UpdatedAt.IsZero() {
		rem.UpdatedAt = now
	}
	if rem.Status == "" {
		rem.Status = StatusActive
	}

	recurrenceJSON, err := encodeRecurrence(rem.Recurrence)
	if err != nil {
		return 0, err
	}

	result, err := r.db.ExecContext(ctx, `
INSERT INTO reminders (
	guild_id, channel_id, creator_user_id, target, message,
	next_run_at, recurrence, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rem.GuildID,
		rem.ChannelID,
		rem.CreatorUserID,
		string(rem.Target),
		rem.Message,
		formatTime(rem.NextRunAt),
		recurrenceJSON,
		string(rem.Status),
		formatTime(rem.CreatedAt),
		formatTime(rem.UpdatedAt),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) ListActiveByCreator(ctx context.Context, guildID, userID string) ([]Reminder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, guild_id, channel_id, creator_user_id, target, message, next_run_at, recurrence, status, created_at, updated_at
FROM reminders
WHERE guild_id = ? AND creator_user_id = ? AND status = ?
ORDER BY next_run_at ASC
LIMIT 20`, guildID, userID, string(StatusActive))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReminders(rows)
}

func (r *Repository) Due(ctx context.Context, now time.Time, limit int) ([]Reminder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, guild_id, channel_id, creator_user_id, target, message, next_run_at, recurrence, status, created_at, updated_at
FROM reminders
WHERE status = ? AND next_run_at <= ?
ORDER BY next_run_at ASC
LIMIT ?`, string(StatusActive), formatTime(now), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReminders(rows)
}

func (r *Repository) CancelByCreator(ctx context.Context, id int64, guildID, userID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE reminders
SET status = ?, updated_at = ?
WHERE id = ? AND guild_id = ? AND creator_user_id = ? AND status = ?`,
		string(StatusCancelled), formatTime(time.Now().UTC()), id, guildID, userID, string(StatusActive))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *Repository) MarkCompleted(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE reminders
SET status = ?, updated_at = ?
WHERE id = ? AND status = ?`,
		string(StatusCompleted), formatTime(time.Now().UTC()), id, string(StatusActive))
	return err
}

func (r *Repository) Reschedule(ctx context.Context, id int64, nextRunAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE reminders
SET next_run_at = ?, updated_at = ?
WHERE id = ? AND status = ?`,
		formatTime(nextRunAt), formatTime(time.Now().UTC()), id, string(StatusActive))
	return err
}

func scanReminders(rows *sql.Rows) ([]Reminder, error) {
	var reminders []Reminder
	for rows.Next() {
		rem, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanReminder(row scanner) (Reminder, error) {
	var rem Reminder
	var target string
	var status string
	var nextRunAt string
	var recurrenceJSON string
	var createdAt string
	var updatedAt string

	if err := row.Scan(
		&rem.ID,
		&rem.GuildID,
		&rem.ChannelID,
		&rem.CreatorUserID,
		&target,
		&rem.Message,
		&nextRunAt,
		&recurrenceJSON,
		&status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Reminder{}, err
	}

	parsedNextRunAt, err := parseStoredTime(nextRunAt)
	if err != nil {
		return Reminder{}, err
	}
	parsedCreatedAt, err := parseStoredTime(createdAt)
	if err != nil {
		return Reminder{}, err
	}
	parsedUpdatedAt, err := parseStoredTime(updatedAt)
	if err != nil {
		return Reminder{}, err
	}
	recurrence, err := decodeRecurrence(recurrenceJSON)
	if err != nil {
		return Reminder{}, err
	}

	rem.Target = Target(target)
	rem.Status = Status(status)
	rem.NextRunAt = parsedNextRunAt
	rem.Recurrence = recurrence
	rem.CreatedAt = parsedCreatedAt
	rem.UpdatedAt = parsedUpdatedAt
	return rem, nil
}

func encodeRecurrence(recurrence *Recurrence) (string, error) {
	if recurrence == nil {
		return "", nil
	}
	data, err := json.Marshal(recurrence)
	if err != nil {
		return "", fmt.Errorf("encode recurrence: %w", err)
	}
	return string(data), nil
}

func decodeRecurrence(data string) (*Recurrence, error) {
	if data == "" {
		return nil, nil
	}
	var recurrence Recurrence
	if err := json.Unmarshal([]byte(data), &recurrence); err != nil {
		return nil, fmt.Errorf("decode recurrence: %w", err)
	}
	return &recurrence, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseStoredTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
