package budget

import(
	"testing"
	"time"
)

func TestEvaluate(t *testing.T){
	// limit #50,000, alert at 80%
	tests := []struct{
		name string
		amount int64
		spent int64
		threshold int
		wantStatus string
		wantPercent int
		wantLeft int64
	}{
		{"nothing spent", 5_000_000, 0, 80, StatusOK, 0, 5_000_000},
		{"well below", 5_000_000, 3_000_000, 80, StatusOK, 60, 2_000_000},
		{"one kobo below threshold", 5_000_000, 3_999_999, 80, StatusOK, 79, 1_000_001},
		{"exactly at threshold", 5_000_000, 4_000_000, 80, StatusWarning, 80, 1_000_000},
		{"between threshold and limit", 5_000_000, 4_999_999, 80, StatusWarning, 99, 1},
		{"exactly at limit", 5_000_000, 5_000_000, 80, StatusLimitReached, 100, 0},
		{"over the limit", 5_000_000, 6_250_000, 80, StatusLimitReached, 125, -1_250_000},
		{"custom threshold 50%", 5_000_000, 2_500_000, 50, StatusWarning, 50, 2_500_000},
		{"threshold 100 only fires at the limit", 5_000_000, 4_999_999, 100, StatusOK, 99, 1},
		{"threshold 100 at the limit", 5_000_000, 5_000_000, 100, StatusLimitReached, 100, 0},
		{"tiny budget below threshold", 3, 2, 67, StatusOK, 66, 1},   // 2/3 = 66.67%, under 67%
		{"tiny budget at threshold", 3, 2, 66, StatusWarning, 66, 1}, // 66.67% is past 66%
	}
	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			b := Budget{Amount: tt.amount, Spent: tt.spent, AlertThreshold: tt.threshold}
			b.Evaluate()
			if b.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", b.Status, tt.wantStatus)
			}
			if b.PercentUsed != tt.wantPercent {
				t.Errorf("percent = %d, want %d", b.PercentUsed, tt.wantPercent)
			}
			if b.Remaining != tt.wantLeft {
				t.Errorf("remaining = %d, want %d", b.Remaining, tt.wantLeft)
			}
		})
	}
}

func TestMonthRange(t *testing.T){
	now := time.Date(2026, 10, 14, 15, 30, 0, 0, time.UTC)

	from, to, label, err := monthRange("", now)
	if err != nil || label != "2026-10" ||
		!from.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) ||
		!to.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("default month: got %v %v %q %v", from, to, label, err)
	}

	// December must roll into January of the next year.
	from, to, label, err = monthRange("2026-12", now)
	if err != nil || label != "2026-12" || !to.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("december: got %v %v %q %v", from, to, label, err)
	}

	for _, bad := range []string{"2026-13", "2026-1", "October", "2026-10-01", "26-10"} {
		if _, _, _, err := monthRange(bad, now); err != ErrInvalidMonth {
			t.Errorf("monthRange(%q) error = %v, want ErrInvalidMonth", bad, err)
		}
	}
}