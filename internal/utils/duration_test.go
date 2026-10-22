package utils_test

import (
	"testing"
	"time"

	"cftestor/internal/utils"
)

func TestParseDurationString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		defaultUnit time.Duration
		want        time.Duration
		wantErr     bool
	}{
		// Pure numbers with default units
		{name: "pure number ms", input: "500", defaultUnit: time.Millisecond, want: 500 * time.Millisecond},
		{name: "pure number s", input: "10", defaultUnit: time.Second, want: 10 * time.Second},
		{name: "pure number min", input: "30", defaultUnit: time.Minute, want: 30 * time.Minute},
		{name: "pure number zero", input: "0", defaultUnit: time.Second, want: 0},
		{name: "pure float number s", input: "1.5", defaultUnit: time.Second, want: 1500 * time.Millisecond},
		{name: "pure leading dot float s", input: ".5", defaultUnit: time.Second, want: 500 * time.Millisecond},
		{name: "pure positive sign", input: "+10", defaultUnit: time.Second, want: 10 * time.Second},

		// Milliseconds
		{name: "50ms", input: "50ms", defaultUnit: time.Second, want: 50 * time.Millisecond},
		{name: "50MS uppercase", input: "50MS", defaultUnit: time.Second, want: 50 * time.Millisecond},
		{name: "50 msec", input: "50 msec", defaultUnit: time.Second, want: 50 * time.Millisecond},
		{name: "50msecs", input: "50msecs", defaultUnit: time.Second, want: 50 * time.Millisecond},
		{name: "50 millisecond", input: "50 millisecond", defaultUnit: time.Second, want: 50 * time.Millisecond},
		{name: "50 milliseconds", input: "50 milliseconds", defaultUnit: time.Second, want: 50 * time.Millisecond},

		// Seconds
		{name: "50s", input: "50s", defaultUnit: time.Millisecond, want: 50 * time.Second},
		{name: ".5s", input: ".5s", defaultUnit: time.Millisecond, want: 500 * time.Millisecond},
		{name: "50sec", input: "50sec", defaultUnit: time.Millisecond, want: 50 * time.Second},
		{name: "50secs", input: "50secs", defaultUnit: time.Millisecond, want: 50 * time.Second},
		{name: "50 second", input: "50 second", defaultUnit: time.Millisecond, want: 50 * time.Second},
		{name: "50 seconds", input: "50 seconds", defaultUnit: time.Millisecond, want: 50 * time.Second},

		// Minutes
		{name: "50m", input: "50m", defaultUnit: time.Second, want: 50 * time.Minute},
		{name: ".5min", input: ".5min", defaultUnit: time.Second, want: 30 * time.Second},
		{name: "50min", input: "50min", defaultUnit: time.Second, want: 50 * time.Minute},
		{name: "50mins", input: "50mins", defaultUnit: time.Second, want: 50 * time.Minute},
		{name: "50 minute", input: "50 minute", defaultUnit: time.Second, want: 50 * time.Minute},
		{name: "50 minutes", input: "50 minutes", defaultUnit: time.Second, want: 50 * time.Minute},

		// Hours, Days, Weeks
		{name: "1h", input: "1h", defaultUnit: time.Minute, want: 1 * time.Hour},
		{name: "2hr", input: "2hr", defaultUnit: time.Minute, want: 2 * time.Hour},
		{name: "3hrs", input: "3hrs", defaultUnit: time.Minute, want: 3 * time.Hour},
		{name: "4 hours", input: "4 hours", defaultUnit: time.Minute, want: 4 * time.Hour},
		{name: "1d", input: "1d", defaultUnit: time.Hour, want: 24 * time.Hour},
		{name: "2 days", input: "2 days", defaultUnit: time.Hour, want: 48 * time.Hour},
		{name: "1w", input: "1w", defaultUnit: time.Hour, want: 7 * 24 * time.Hour},
		{name: "2 weeks", input: "2 weeks", defaultUnit: time.Hour, want: 14 * 24 * time.Hour},

		// Compound durations
		{name: "1m30s", input: "1m30s", defaultUnit: time.Second, want: 90 * time.Second},
		{name: "1h 30m", input: "1h 30m", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1h+30m", input: "1h+30m", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1h + 30m", input: "1h + 30m", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1 hour, 30 mins", input: "1 hour, 30 mins", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1 hour and 30 minutes", input: "1 hour and 30 minutes", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1 hour, + 30 mins", input: "1 hour, + 30 mins", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "1h, and 30m", input: "1h, and 30m", defaultUnit: time.Minute, want: 90 * time.Minute},
		{name: "500ms 2s", input: "500ms 2s", defaultUnit: time.Millisecond, want: 2500 * time.Millisecond},

		// Microseconds & Nanoseconds
		{name: "500us", input: "500us", defaultUnit: time.Millisecond, want: 500 * time.Microsecond},
		{name: "500usec", input: "500usec", defaultUnit: time.Millisecond, want: 500 * time.Microsecond},
		{name: "500usecs", input: "500usecs", defaultUnit: time.Millisecond, want: 500 * time.Microsecond},
		{name: "500µs micro sign", input: "500µs", defaultUnit: time.Millisecond, want: 500 * time.Microsecond},
		{name: "500μs greek mu", input: "500μs", defaultUnit: time.Millisecond, want: 500 * time.Microsecond},
		{name: "1000ns", input: "1000ns", defaultUnit: time.Millisecond, want: 1000 * time.Nanosecond},
		{name: "1000nsec", input: "1000nsec", defaultUnit: time.Millisecond, want: 1000 * time.Nanosecond},

		// Positive / Negative signs
		{name: "positive 50ms", input: "+50ms", defaultUnit: time.Millisecond, want: 50 * time.Millisecond},
		{name: "negative 50ms", input: "-50ms", defaultUnit: time.Millisecond, want: -50 * time.Millisecond},
		{name: "negative pure number", input: "-10", defaultUnit: time.Second, want: -10 * time.Second},

		// Error cases
		{name: "empty string", input: "", defaultUnit: time.Second, wantErr: true},
		{name: "spaces only", input: "   ", defaultUnit: time.Second, wantErr: true},
		{name: "invalid text", input: "abc", defaultUnit: time.Second, wantErr: true},
		{name: "unknown unit", input: "50foo", defaultUnit: time.Second, wantErr: true},
		{name: "number after unit without unit", input: "50ms 10", defaultUnit: time.Millisecond, wantErr: true},
		{name: "hex duration number", input: "0x10", defaultUnit: time.Second, wantErr: true},
		{name: "hex duration with unit", input: "0x10s", defaultUnit: time.Second, wantErr: true},
		{name: "double minus sign", input: "--50", defaultUnit: time.Millisecond, wantErr: true},
		{name: "minus plus sign", input: "-+50", defaultUnit: time.Millisecond, wantErr: true},
		{name: "plus minus sign", input: "+-50", defaultUnit: time.Millisecond, wantErr: true},
		{name: "double plus sign", input: "++50", defaultUnit: time.Millisecond, wantErr: true},
		{name: "double minus sign with unit", input: "--50ms", defaultUnit: time.Millisecond, wantErr: true},
		{name: "overflow duration number", input: "999999999999999999999999999999999999999999", defaultUnit: time.Hour, wantErr: true},
		{name: "overflow duration with unit", input: "999999999999999999999999999999999999999999h", defaultUnit: time.Hour, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.ParseDurationString(tt.input, tt.defaultUnit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDurationString(%q, %v) error = %v, wantErr %v", tt.input, tt.defaultUnit, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("ParseDurationString(%q, %v) = %v, want %v", tt.input, tt.defaultUnit, got, tt.want)
			}
		})
	}
}
