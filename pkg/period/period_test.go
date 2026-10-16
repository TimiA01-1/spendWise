package period

import (
	"testing"
	"time"
)

func TestMonth(t *testing.T) {
	now := time.Date(2026, 10, 14, 15, 30, 0, 0, time.UTC)

	from, to, label, err := Month("", now)
	if err != nil || label != "2026-10" ||
		!from.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) ||
		!to.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("default month: got %v %v %q %v", from, to, label, err)
	}

	// December must roll into January of the next year.
	_, to, label, err = Month("2026-12", now)
	if err != nil || label != "2026-12" || !to.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("december: got %v %q %v", to, label, err)
	}

	// February in a leap year ends on the 29th, so March 1st is the exclusive end.
	_, to, _, _ = Month("2028-02", now)
	if !to.Equal(time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("leap february: to = %v", to)
	}

	for _, bad := range []string{"2026-13", "2026-1", "October", "2026-10-01", "26-10"} {
		if _, _, _, err := Month(bad, now); err != ErrInvalidMonth {
			t.Errorf("Month(%q) error = %v, want ErrInvalidMonth", bad, err)
		}
	}
}