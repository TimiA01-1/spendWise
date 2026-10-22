package recurring

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func TestOccurrence_Monthly(t *testing.T){
	start := d(2026, 1, 31)
	want := []time.Time{
		d(2026, 1, 31), d(2026, 2, 28), d(2026, 3, 31), d(2026, 4, 30),
		d(2026, 5, 31), d(2026, 6, 30), d(2026, 7, 31), d(2026, 8, 31),
		d(2026, 9, 30), d(2026, 10, 31), d(2026, 11, 30), d(2026, 12, 31),
		d(2027, 1, 31), // wraps into the next year
	}
	for n, w := range want {
		if got := Occurrence(start, FreqMonthly, n); !got.Equal(w) {
			t.Errorf("monthly n=%d: got %s, want %s", n, got.Format(dateLayout), w.Format(dateLayout))
		}
	}

	// Leap year: February has 29 days in 2028.
	if got := Occurrence(d(2028, 1, 31), FreqMonthly, 1); !got.Equal(d(2028, 2, 29)) {
		t.Errorf("leap february: got %s", got.Format(dateLayout))
	}

	// An ordinary day of the month never moves.
	for n := 0; n < 30; n++ {
		if got := Occurrence(d(2026, 1, 15), FreqMonthly, n); got.Day() != 15 {
			t.Fatalf("15th drifted at n=%d: %s", n, got.Format(dateLayout))
		}
	}
}

func TestOccurrence_OtherFrequencies(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		freq  string
		n int
		want  time.Time
	}{
		{"n=0 is the start date", d(2026, 3, 9), FreqMonthly, 0, d(2026, 3, 9)},
		{"daily across a month end", d(2026, 1, 30), FreqDaily, 3, d(2026, 2, 2)},
		{"daily across a leap day", d(2028, 2, 28), FreqDaily, 2, d(2028, 3, 1)},
		{"weekly", d(2026, 1, 1), FreqWeekly, 5, d(2026, 2, 5)},
		{"yearly", d(2026, 6, 15), FreqYearly, 3, d(2029, 6, 15)},
		{"yearly on Feb 29: non-leap year clamps", d(2028, 2, 29), FreqYearly, 1, d(2029, 2, 28)},
		{"yearly on Feb 29: back to the 29th in a leap year", d(2028, 2, 29), FreqYearly, 4, d(2032, 2, 29)},
		{"time of day is ignored", time.Date(2026, 5, 5, 23, 59, 0, 0, time.UTC), FreqDaily, 1, d(2026, 5, 6)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Occurrence(tt.start, tt.freq, tt.n); !got.Equal(tt.want) {
				t.Errorf("got %s, want %s", got.Format(dateLayout), tt.want.Format(dateLayout))
			}
		})
	}
}

func TestFirstIndexOnOrAfter(t *testing.T) {
	start := d(2026, 1, 31) // monthly: Jan31 Feb28 Mar31 Apr30 May31 Jun30 Jul31 ...

	tests := []struct {
		name string
		from int
		day  time.Time
		want int
	}{
		{"already in the future: nothing skipped", 3, d(2026, 4, 1), 3},
		{"exactly on an occurrence: it is NOT skipped", 3, d(2026, 4, 30), 3},
		{"skips the ones inside the pause", 3, d(2026, 7, 10), 6}, // Apr30, May31, Jun30 skipped -> Jul31
		{"one day after an occurrence skips it", 3, d(2026, 5, 1), 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstIndexOnOrAfter(start, FreqMonthly, tt.from, tt.day); got != tt.want {
				t.Errorf("index = %d, want %d", got, tt.want)
			}
		})
	}
}
