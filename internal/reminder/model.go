package reminder

import (
	"fmt"
	"time"
)

type Target string

const (
	TargetMe      Target = "me"
	TargetChannel Target = "channel"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type RecurrenceKind string

const (
	RecurrenceDaily   RecurrenceKind = "daily"
	RecurrenceWeekday RecurrenceKind = "weekday"
	RecurrenceWeekly  RecurrenceKind = "weekly"
)

type Recurrence struct {
	Kind     RecurrenceKind `json:"kind"`
	Weekday  *int           `json:"weekday,omitempty"`
	Hour     int            `json:"hour"`
	Minute   int            `json:"minute"`
	Timezone string         `json:"timezone"`
}

type Reminder struct {
	ID            int64
	GuildID       string
	ChannelID     string
	CreatorUserID string
	Target        Target
	Message       string
	NextRunAt     time.Time
	Recurrence    *Recurrence
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Parsed struct {
	Message    string
	NextRunAt  time.Time
	Recurrence *Recurrence
}

func (r Recurrence) NextAfter(after time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load recurrence timezone %q: %w", r.Timezone, err)
	}

	localAfter := after.In(loc)
	switch r.Kind {
	case RecurrenceDaily:
		return nextDaily(localAfter, r.Hour, r.Minute), nil
	case RecurrenceWeekday:
		return nextMatchingDay(localAfter, r.Hour, r.Minute, func(d time.Weekday) bool {
			return d >= time.Monday && d <= time.Friday
		}), nil
	case RecurrenceWeekly:
		if r.Weekday == nil || *r.Weekday < int(time.Sunday) || *r.Weekday > int(time.Saturday) {
			return time.Time{}, fmt.Errorf("weekly recurrence requires weekday")
		}
		weekday := time.Weekday(*r.Weekday)
		return nextMatchingDay(localAfter, r.Hour, r.Minute, func(d time.Weekday) bool {
			return d == weekday
		}), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported recurrence kind %q", r.Kind)
	}
}

func nextDaily(after time.Time, hour, minute int) time.Time {
	candidate := time.Date(after.Year(), after.Month(), after.Day(), hour, minute, 0, 0, after.Location())
	if !candidate.After(after) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

func nextMatchingDay(after time.Time, hour, minute int, matches func(time.Weekday) bool) time.Time {
	for offset := 0; offset <= 7; offset++ {
		day := after.AddDate(0, 0, offset)
		if !matches(day.Weekday()) {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, after.Location())
		if candidate.After(after) {
			return candidate
		}
	}
	return time.Time{}
}
