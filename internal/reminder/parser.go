package reminder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type parseMatch struct {
	start      int
	end        int
	nextRunAt  time.Time
	recurrence *Recurrence
}

var (
	absoluteDateRe   = regexp.MustCompile(`(?i)([0-9０-９]{4})[-/／]([0-9０-９]{1,2})[-/／]([0-9０-９]{1,2})(?:[\s　T]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)
	japaneseDateRe   = regexp.MustCompile(`([0-9０-９]{4})[\s　]*年[\s　]*([0-9０-９]{1,2})[\s　]*月[\s　]*([0-9０-９]{1,2})[\s　]*日(?:[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	japaneseRelative = regexp.MustCompile(`([0-9０-９]+)[\s　]*(分|時間|日|週間)後`)
	englishRelative  = regexp.MustCompile(`(?i)\bin[\s　]+([0-9０-９]+)[\s　]*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days|w|week|weeks)\b`)
	japaneseTomorrow = regexp.MustCompile(`明日(?:[\s　]*(?:の)?[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	englishTomorrow  = regexp.MustCompile(`(?i)\btomorrow(?:[\s　]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)
	japaneseWeekday  = regexp.MustCompile(`(?:来週[\s　]*)?(月曜(?:日)?|火曜(?:日)?|水曜(?:日)?|木曜(?:日)?|金曜(?:日)?|土曜(?:日)?|日曜(?:日)?)(?:[\s　]*(?:の)?[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	englishWeekday   = regexp.MustCompile(`(?i)\b(?:next[\s　]+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:[\s　]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)

	japaneseDailyRecurrence   = regexp.MustCompile(`毎日(?:[\s　]*(?:の)?[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	japaneseWeekdayRecurrence = regexp.MustCompile(`(?:毎)?平日(?:[\s　]*(?:の)?[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	japaneseWeeklyRecurrence  = regexp.MustCompile(`毎週[\s　]*(月曜(?:日)?|火曜(?:日)?|水曜(?:日)?|木曜(?:日)?|金曜(?:日)?|土曜(?:日)?|日曜(?:日)?)(?:[\s　]*(?:の)?[\s　]*(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}[\s　]*時(?:[\s　]*[0-9０-９]{1,2}[\s　]*分?)?))?(?:[\s　]*に)?`)
	englishDailyRecurrence    = regexp.MustCompile(`(?i)\b(?:every[\s　]+day|daily)(?:[\s　]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)
	englishWeekdayRecurrence  = regexp.MustCompile(`(?i)\bevery[\s　]+weekdays?(?:[\s　]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)
	englishWeeklyRecurrence   = regexp.MustCompile(`(?i)\bevery[\s　]+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:[\s　]+(?:at[\s　]+)?(?:[0-9０-９]{1,2}[:：][0-9０-９]{1,2}|[0-9０-９]{1,2}(?:[:：][0-9０-９]{1,2})?[\s　]*(?:am|pm)))?`)

	colonTimeRe = regexp.MustCompile(`([0-9０-９]{1,2})[:：]([0-9０-９]{1,2})`)
	jpTimeRe    = regexp.MustCompile(`([0-9０-９]{1,2})[\s　]*時(?:[\s　]*([0-9０-９]{1,2})[\s　]*分?)?`)
	ampmTimeRe  = regexp.MustCompile(`(?i)\b([0-9０-９]{1,2})(?:[:：]([0-9０-９]{1,2}))?[\s　]*(am|pm)\b`)
	spacesRe    = regexp.MustCompile(`[ \t\r\n　]+`)
)

func Parse(input string, now time.Time, loc *time.Location) (Parsed, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Parsed{}, fmt.Errorf("リマインダー本文を入力してください")
	}
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)

	matchers := []func(string, time.Time, *time.Location) (parseMatch, bool, error){
		matchRecurring,
		matchRelative,
		matchAbsolute,
		matchTomorrow,
		matchWeekday,
	}
	for _, matcher := range matchers {
		match, ok, err := matcher(input, now, loc)
		if err != nil {
			return Parsed{}, err
		}
		if !ok {
			continue
		}

		message := cleanupMessage(input, match.start, match.end)
		if message == "" {
			return Parsed{}, fmt.Errorf("日時以外のリマインド本文を入力してください")
		}
		if !match.nextRunAt.After(now) {
			return Parsed{}, fmt.Errorf("未来の日時を指定してください")
		}
		return Parsed{
			Message:    message,
			NextRunAt:  match.nextRunAt,
			Recurrence: match.recurrence,
		}, nil
	}

	return Parsed{}, fmt.Errorf("日時を解釈できませんでした")
}

func matchRecurring(input string, now time.Time, loc *time.Location) (parseMatch, bool, error) {
	if m, ok, err := matchRecurrence(input, now, loc, japaneseWeeklyRecurrence, RecurrenceWeekly); ok || err != nil {
		return m, ok, err
	}
	if m, ok, err := matchRecurrence(input, now, loc, englishWeeklyRecurrence, RecurrenceWeekly); ok || err != nil {
		return m, ok, err
	}
	if m, ok, err := matchRecurrence(input, now, loc, japaneseWeekdayRecurrence, RecurrenceWeekday); ok || err != nil {
		return m, ok, err
	}
	if m, ok, err := matchRecurrence(input, now, loc, englishWeekdayRecurrence, RecurrenceWeekday); ok || err != nil {
		return m, ok, err
	}
	if m, ok, err := matchRecurrence(input, now, loc, japaneseDailyRecurrence, RecurrenceDaily); ok || err != nil {
		return m, ok, err
	}
	return matchRecurrence(input, now, loc, englishDailyRecurrence, RecurrenceDaily)
}

func matchRecurrence(input string, now time.Time, loc *time.Location, re *regexp.Regexp, kind RecurrenceKind) (parseMatch, bool, error) {
	index := re.FindStringIndex(input)
	if index == nil {
		return parseMatch{}, false, nil
	}
	segment := input[index[0]:index[1]]
	hour, minute, _, err := parseClock(segment, 9, 0)
	if err != nil {
		return parseMatch{}, false, err
	}

	recurrence := &Recurrence{
		Kind:     kind,
		Hour:     hour,
		Minute:   minute,
		Timezone: loc.String(),
	}
	if kind == RecurrenceWeekly {
		weekday, ok := weekdayFromText(segment)
		if !ok {
			return parseMatch{}, false, fmt.Errorf("曜日を解釈できませんでした")
		}
		w := int(weekday)
		recurrence.Weekday = &w
	}

	next, err := recurrence.NextAfter(now)
	if err != nil {
		return parseMatch{}, false, err
	}
	return parseMatch{start: index[0], end: index[1], nextRunAt: next, recurrence: recurrence}, true, nil
}

func matchRelative(input string, now time.Time, loc *time.Location) (parseMatch, bool, error) {
	if match := japaneseRelative.FindStringSubmatchIndex(input); match != nil {
		n, unit, err := relativeParts(input, match)
		if err != nil {
			return parseMatch{}, false, err
		}
		duration, err := japaneseDuration(n, unit)
		if err != nil {
			return parseMatch{}, false, err
		}
		return parseMatch{start: match[0], end: match[1], nextRunAt: now.Add(duration)}, true, nil
	}

	if match := englishRelative.FindStringSubmatchIndex(input); match != nil {
		n, unit, err := relativeParts(input, match)
		if err != nil {
			return parseMatch{}, false, err
		}
		duration, err := englishDuration(n, unit)
		if err != nil {
			return parseMatch{}, false, err
		}
		return parseMatch{start: match[0], end: match[1], nextRunAt: now.Add(duration)}, true, nil
	}

	return parseMatch{}, false, nil
}

func matchAbsolute(input string, now time.Time, loc *time.Location) (parseMatch, bool, error) {
	if match := absoluteDateRe.FindStringSubmatchIndex(input); match != nil {
		next, err := parseAbsoluteDate(input, match, loc)
		if err != nil {
			return parseMatch{}, false, err
		}
		return parseMatch{start: match[0], end: match[1], nextRunAt: next}, true, nil
	}
	if match := japaneseDateRe.FindStringSubmatchIndex(input); match != nil {
		next, err := parseAbsoluteDate(input, match, loc)
		if err != nil {
			return parseMatch{}, false, err
		}
		return parseMatch{start: match[0], end: match[1], nextRunAt: next}, true, nil
	}
	return parseMatch{}, false, nil
}

func matchTomorrow(input string, now time.Time, loc *time.Location) (parseMatch, bool, error) {
	if index := japaneseTomorrow.FindStringIndex(input); index != nil {
		segment := input[index[0]:index[1]]
		hour, minute, _, err := parseClock(segment, 9, 0)
		if err != nil {
			return parseMatch{}, false, err
		}
		day := now.AddDate(0, 0, 1)
		next := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
		return parseMatch{start: index[0], end: index[1], nextRunAt: next}, true, nil
	}
	if index := englishTomorrow.FindStringIndex(input); index != nil {
		segment := input[index[0]:index[1]]
		hour, minute, _, err := parseClock(segment, 9, 0)
		if err != nil {
			return parseMatch{}, false, err
		}
		day := now.AddDate(0, 0, 1)
		next := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
		return parseMatch{start: index[0], end: index[1], nextRunAt: next}, true, nil
	}
	return parseMatch{}, false, nil
}

func matchWeekday(input string, now time.Time, loc *time.Location) (parseMatch, bool, error) {
	if index := japaneseWeekday.FindStringIndex(input); index != nil {
		segment := input[index[0]:index[1]]
		weekday, ok := weekdayFromText(segment)
		if !ok {
			return parseMatch{}, false, fmt.Errorf("曜日を解釈できませんでした")
		}
		hour, minute, _, err := parseClock(segment, 9, 0)
		if err != nil {
			return parseMatch{}, false, err
		}
		next := nextWeekday(now, hour, minute, weekday, strings.Contains(segment, "来週"))
		return parseMatch{start: index[0], end: index[1], nextRunAt: next}, true, nil
	}
	if index := englishWeekday.FindStringIndex(input); index != nil {
		segment := input[index[0]:index[1]]
		weekday, ok := weekdayFromText(segment)
		if !ok {
			return parseMatch{}, false, fmt.Errorf("曜日を解釈できませんでした")
		}
		hour, minute, _, err := parseClock(segment, 9, 0)
		if err != nil {
			return parseMatch{}, false, err
		}
		next := nextWeekday(now, hour, minute, weekday, strings.HasPrefix(strings.ToLower(strings.TrimSpace(segment)), "next"))
		return parseMatch{start: index[0], end: index[1], nextRunAt: next}, true, nil
	}
	return parseMatch{}, false, nil
}

func nextWeekday(now time.Time, hour, minute int, weekday time.Weekday, forceNext bool) time.Time {
	start := now
	if forceNext {
		start = start.AddDate(0, 0, 1)
	}
	return nextMatchingDay(start, hour, minute, func(d time.Weekday) bool {
		return d == weekday
	})
}

func parseAbsoluteDate(input string, match []int, loc *time.Location) (time.Time, error) {
	year, err := atoiDigits(input[match[2]:match[3]])
	if err != nil {
		return time.Time{}, err
	}
	month, err := atoiDigits(input[match[4]:match[5]])
	if err != nil {
		return time.Time{}, err
	}
	day, err := atoiDigits(input[match[6]:match[7]])
	if err != nil {
		return time.Time{}, err
	}

	segment := input[match[0]:match[1]]
	hour, minute, _, err := parseClock(segment, 9, 0)
	if err != nil {
		return time.Time{}, err
	}
	next := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	if next.Year() != year || int(next.Month()) != month || next.Day() != day {
		return time.Time{}, fmt.Errorf("存在しない日付です")
	}
	return next, nil
}

func relativeParts(input string, match []int) (int, string, error) {
	n, err := atoiDigits(input[match[2]:match[3]])
	if err != nil {
		return 0, "", err
	}
	if n <= 0 {
		return 0, "", fmt.Errorf("相対日時は1以上で指定してください")
	}
	return n, strings.ToLower(input[match[4]:match[5]]), nil
}

func japaneseDuration(n int, unit string) (time.Duration, error) {
	switch unit {
	case "分":
		return time.Duration(n) * time.Minute, nil
	case "時間":
		return time.Duration(n) * time.Hour, nil
	case "日":
		return time.Duration(n) * 24 * time.Hour, nil
	case "週間":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("未対応の相対日時単位です")
	}
}

func englishDuration(n int, unit string) (time.Duration, error) {
	switch unit {
	case "m", "min", "mins", "minute", "minutes":
		return time.Duration(n) * time.Minute, nil
	case "h", "hr", "hrs", "hour", "hours":
		return time.Duration(n) * time.Hour, nil
	case "d", "day", "days":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w", "week", "weeks":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported relative time unit")
	}
}

func parseClock(segment string, defaultHour, defaultMinute int) (int, int, bool, error) {
	if m := colonTimeRe.FindStringSubmatch(segment); m != nil {
		hour, _ := atoiDigits(m[1])
		minute, _ := atoiDigits(m[2])
		if hour > 23 || minute > 59 {
			return 0, 0, false, fmt.Errorf("時刻を解釈できませんでした")
		}
		return hour, minute, true, nil
	}
	if m := jpTimeRe.FindStringSubmatch(segment); m != nil {
		hour, _ := atoiDigits(m[1])
		minute := 0
		if m[2] != "" {
			minute, _ = atoiDigits(m[2])
		}
		if hour > 23 || minute > 59 {
			return 0, 0, false, fmt.Errorf("時刻を解釈できませんでした")
		}
		return hour, minute, true, nil
	}
	if m := ampmTimeRe.FindStringSubmatch(segment); m != nil {
		hour, _ := atoiDigits(m[1])
		minute := 0
		if m[2] != "" {
			minute, _ = atoiDigits(m[2])
		}
		if hour < 1 || hour > 12 || minute > 59 {
			return 0, 0, false, fmt.Errorf("時刻を解釈できませんでした")
		}
		period := strings.ToLower(m[3])
		if period == "pm" && hour != 12 {
			hour += 12
		}
		if period == "am" && hour == 12 {
			hour = 0
		}
		return hour, minute, true, nil
	}
	return defaultHour, defaultMinute, false, nil
}

func atoiDigits(value string) (int, error) {
	return strconv.Atoi(normalizeDigits(value))
}

func normalizeDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r >= '０' && r <= '９' {
			b.WriteRune('0' + (r - '０'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func weekdayFromText(text string) (time.Weekday, bool) {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "日曜") || strings.Contains(lower, "sunday"):
		return time.Sunday, true
	case strings.Contains(lower, "月曜") || strings.Contains(lower, "monday"):
		return time.Monday, true
	case strings.Contains(lower, "火曜") || strings.Contains(lower, "tuesday"):
		return time.Tuesday, true
	case strings.Contains(lower, "水曜") || strings.Contains(lower, "wednesday"):
		return time.Wednesday, true
	case strings.Contains(lower, "木曜") || strings.Contains(lower, "thursday"):
		return time.Thursday, true
	case strings.Contains(lower, "金曜") || strings.Contains(lower, "friday"):
		return time.Friday, true
	case strings.Contains(lower, "土曜") || strings.Contains(lower, "saturday"):
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func cleanupMessage(input string, start, end int) string {
	message := strings.TrimSpace(input[:start] + " " + input[end:])
	message = strings.Trim(message, " \t\r\n　,，.。")
	message = spacesRe.ReplaceAllString(message, " ")
	message = strings.TrimSpace(message)

	for {
		before := message
		message = strings.TrimPrefix(message, "に")
		message = strings.TrimPrefix(message, "を")
		message = strings.TrimPrefix(message, "to ")
		message = strings.TrimPrefix(message, "at ")
		message = strings.TrimSpace(message)
		message = strings.TrimSuffix(message, "に")
		message = strings.TrimSuffix(message, "を")
		message = strings.TrimSuffix(message, " at")
		message = strings.TrimSuffix(message, " on")
		message = strings.Trim(message, " \t\r\n　,，.。")
		if message == before {
			break
		}
	}
	return message
}
