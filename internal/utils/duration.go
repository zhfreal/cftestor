package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// durationSegmentPattern matches a number (integer or decimal, including leading dot like .5) followed by unit letters.
	durationSegmentPattern = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?|\.[0-9]+)\s*(\p{L}+)`)
)

// ParseDurationString parses user-provided duration strings supporting natural units
// (e.g. 50ms, 50s, 50sec, 50secs, 50m, 50min, 50mins, 50minutes, 1h, 1hr, 2hours, 1d, 1w)
// and compound durations (e.g. 1m30s, 1h15m, 2h 30m, 1h+30m), as well as plain numbers without
// units by applying defaultUnit.
func ParseDurationString(raw string, defaultUnit time.Duration) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	isNegative := false
	if strings.HasPrefix(s, "-") {
		isNegative = true
		s = strings.TrimSpace(s[1:])
	} else if strings.HasPrefix(s, "+") {
		s = strings.TrimSpace(s[1:])
	}

	if s == "" {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}

	// Reject redundant / multiple signs such as "--50", "-+50", "++50".
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		return 0, fmt.Errorf("invalid duration %q: multiple signs", raw)
	}

	// Reject hexadecimal numbers such as "0x10"
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		return 0, fmt.Errorf("invalid duration number %q", raw)
	}

	// 1. Check if raw is a pure number without unit.
	if num, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(num) || math.IsInf(num, 0) {
			return 0, fmt.Errorf("invalid duration number %q", raw)
		}
		if defaultUnit <= 0 {
			defaultUnit = time.Millisecond
		}
		durFloat := num * float64(defaultUnit)
		if durFloat >= float64(math.MaxInt64) || durFloat < 0 {
			return 0, fmt.Errorf("duration %q overflows maximum supported duration", raw)
		}
		dur := time.Duration(durFloat)
		if isNegative {
			dur = -dur
		}
		return dur, nil
	}

	// 2. Parse unit-based segments (handles single or compound durations like "1h30m", "50mins", "500 ms", "1h+30m", "1 hour, + 30 mins").
	var totalDur time.Duration
	cur := s
	matchedAny := false

	for len(cur) > 0 {
		// Skip optional commas, pluses, spaces, or 'and' between segments
		for {
			cur = strings.TrimSpace(cur)
			if strings.HasPrefix(cur, ",") {
				cur = cur[1:]
			} else if strings.HasPrefix(cur, "+") {
				cur = cur[1:]
			} else if len(cur) >= 4 && strings.EqualFold(cur[:4], "and ") {
				cur = cur[4:]
			} else {
				break
			}
		}

		if cur == "" {
			break
		}

		loc := durationSegmentPattern.FindStringSubmatchIndex(cur)
		if loc == nil {
			return 0, fmt.Errorf("invalid duration format %q", raw)
		}

		numStr := cur[loc[2]:loc[3]]
		unitStr := cur[loc[4]:loc[5]]
		cur = cur[loc[1]:]

		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
			return 0, fmt.Errorf("invalid number %q in duration %q", numStr, raw)
		}

		unitMultiplier, err := parseDurationUnit(unitStr)
		if err != nil {
			return 0, fmt.Errorf("%w in duration %q", err, raw)
		}

		segmentFloat := val * float64(unitMultiplier)
		if segmentFloat >= float64(math.MaxInt64) || segmentFloat < 0 {
			return 0, fmt.Errorf("duration segment %q overflows maximum supported duration", numStr+unitStr)
		}

		segmentDur := time.Duration(segmentFloat)
		if totalDur > time.Duration(math.MaxInt64)-segmentDur {
			return 0, fmt.Errorf("duration %q overflows maximum supported duration", raw)
		}

		totalDur += segmentDur
		matchedAny = true
	}

	if !matchedAny {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}

	if isNegative {
		totalDur = -totalDur
	}
	return totalDur, nil
}

func parseDurationUnit(u string) (time.Duration, error) {
	switch strings.ToLower(u) {
	case "ns", "nsec", "nsecs", "nanosecond", "nanoseconds":
		return time.Nanosecond, nil
	case "us", "µs", "μs", "usec", "usecs", "microsecond", "microseconds":
		return time.Microsecond, nil
	case "ms", "msec", "msecs", "millisecond", "milliseconds":
		return time.Millisecond, nil
	case "s", "sec", "secs", "second", "seconds":
		return time.Second, nil
	case "m", "min", "mins", "minute", "minutes":
		return time.Minute, nil
	case "h", "hr", "hrs", "hour", "hours":
		return time.Hour, nil
	case "d", "day", "days":
		return 24 * time.Hour, nil
	case "w", "wk", "wks", "week", "weeks":
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown time unit %q", u)
	}
}
