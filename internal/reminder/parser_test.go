package reminder

import (
	"testing"
	"time"
)

func TestParseSupportedSyntax(t *testing.T) {
	loc := mustLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, loc)

	tests := []struct {
		name       string
		input      string
		wantMsg    string
		wantTime   time.Time
		wantRepeat RecurrenceKind
	}{
		{
			name:     "japanese tomorrow",
			input:    "明日9時に請求書確認",
			wantMsg:  "請求書確認",
			wantTime: time.Date(2026, 5, 26, 9, 0, 0, 0, loc),
		},
		{
			name:     "japanese relative",
			input:    "10分後に水を飲む",
			wantMsg:  "水を飲む",
			wantTime: now.Add(10 * time.Minute),
		},
		{
			name:     "full-width japanese relative",
			input:    "１０分後に水を飲む",
			wantMsg:  "水を飲む",
			wantTime: now.Add(10 * time.Minute),
		},
		{
			name:     "absolute date",
			input:    "請求書確認 2026-05-26 09:30",
			wantMsg:  "請求書確認",
			wantTime: time.Date(2026, 5, 26, 9, 30, 0, 0, loc),
		},
		{
			name:     "full-width absolute date and colon",
			input:    "請求書確認　２０２６／０５／２６　０９：３０",
			wantMsg:  "請求書確認",
			wantTime: time.Date(2026, 5, 26, 9, 30, 0, 0, loc),
		},
		{
			name:       "japanese daily recurrence",
			input:      "毎日9時に日報",
			wantMsg:    "日報",
			wantTime:   time.Date(2026, 5, 25, 9, 0, 0, 0, loc),
			wantRepeat: RecurrenceDaily,
		},
		{
			name:       "english weekly recurrence",
			input:      "every monday 10am standup",
			wantMsg:    "standup",
			wantTime:   time.Date(2026, 5, 25, 10, 0, 0, 0, loc),
			wantRepeat: RecurrenceWeekly,
		},
		{
			name:       "japanese weekday recurrence",
			input:      "平日9時に朝会",
			wantMsg:    "朝会",
			wantTime:   time.Date(2026, 5, 25, 9, 0, 0, 0, loc),
			wantRepeat: RecurrenceWeekday,
		},
		{
			name:       "full-width spaces and time",
			input:      "毎週　月曜　１０：３０　に　朝会",
			wantMsg:    "朝会",
			wantTime:   time.Date(2026, 5, 25, 10, 30, 0, 0, loc),
			wantRepeat: RecurrenceWeekly,
		},
		{
			name:     "english tomorrow",
			input:    "tomorrow 9am submit report",
			wantMsg:  "submit report",
			wantTime: time.Date(2026, 5, 26, 9, 0, 0, 0, loc),
		},
		{
			name:     "japanese next weekday",
			input:    "来週月曜10時に定例",
			wantMsg:  "定例",
			wantTime: time.Date(2026, 6, 1, 10, 0, 0, 0, loc),
		},
		{
			name:     "english next weekday",
			input:    "next monday 10am meeting",
			wantMsg:  "meeting",
			wantTime: time.Date(2026, 6, 1, 10, 0, 0, 0, loc),
		},
		{
			name:     "japanese word starting with particle char preserved",
			input:    "明日9時ににんじん買う",
			wantMsg:  "にんじん買う",
			wantTime: time.Date(2026, 5, 26, 9, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, now, loc)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if !got.NextRunAt.Equal(tt.wantTime) {
				t.Fatalf("NextRunAt = %s, want %s", got.NextRunAt, tt.wantTime)
			}
			if tt.wantRepeat == "" {
				if got.Recurrence != nil {
					t.Fatalf("Recurrence = %#v, want nil", got.Recurrence)
				}
				return
			}
			if got.Recurrence == nil || got.Recurrence.Kind != tt.wantRepeat {
				t.Fatalf("Recurrence = %#v, want kind %s", got.Recurrence, tt.wantRepeat)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	loc := mustLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 5, 25, 8, 0, 0, 0, loc)

	tests := []string{
		"日時なしのタスク",
		"2026-05-24 09:00 過去の予定",
		"明日9時",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input, now, loc); err == nil {
				t.Fatalf("Parse(%q) expected error", input)
			}
		})
	}
}

func TestRecurrenceNextAfter(t *testing.T) {
	loc := mustLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, loc)

	monday := int(time.Monday)
	recurrence := Recurrence{
		Kind:     RecurrenceWeekly,
		Weekday:  &monday,
		Hour:     9,
		Minute:   30,
		Timezone: loc.String(),
	}

	got, err := recurrence.NextAfter(now)
	if err != nil {
		t.Fatalf("NextAfter() error = %v", err)
	}
	want := time.Date(2026, 6, 1, 9, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextAfter() = %s, want %s", got, want)
	}
}

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
