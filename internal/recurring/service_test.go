package recurring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"spendWise/internal/transaction"
)

func baseRule() Rule {
	return Rule{
		ID: uuid.New(),
		Type: transaction.TypeExpense,
		Amount: 15_000_000,
		Frequency: FreqMonthly,
		StartDate: d(2026, 1, 31),
		NextIndex: 3, // Apr 30 is next
		NextRunOn: d(2026, 4, 30),
		Status: StatusActive,
	}
}

func TestApplyUpdate(t *testing.T){
	today := d(2026, 7, 10)
	str := func(s string) *string { return &s }
	i64 := func(n int64) *int64 { return &n }
	date := func(t time.Time) *time.Time { return &t }

	t.Run("amount and note change; schedule untouched", func(t *testing.T) {
		got, err := applyUpdate(baseRule(), UpdateInput{Amount: i64(20_000_000), Note: str("  New rent ")}, today)
		if err != nil || got.Amount != 20_000_000 || got.Note != "New rent" ||
			got.NextIndex != 3 || !got.NextRunOn.Equal(d(2026, 4, 30)) || got.Status != StatusActive {
			t.Errorf("got %+v, err %v", got, err)
		}
	})

	t.Run("pause keeps the schedule where it is", func(t *testing.T) {
		got, _ := applyUpdate(baseRule(), UpdateInput{Status: str(StatusPaused)}, today)
		if got.Status != StatusPaused || got.NextIndex != 3 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("resume skips the occurrences that fell inside the pause", func(t *testing.T) {
		paused := baseRule()
		paused.Status = StatusPaused
		got, _ := applyUpdate(paused, UpdateInput{Status: str(StatusActive)}, today)
		// Apr 30, May 31 and Jun 30 are skipped; the next one is Jul 31.
		if got.Status != StatusActive || got.NextIndex != 6 || !got.NextRunOn.Equal(d(2026, 7, 31)) {
			t.Errorf("got status=%s index=%d next=%s", got.Status, got.NextIndex, got.NextRunOn.Format(dateLayout))
		}
	})

	t.Run("activating a rule that is already active changes nothing", func(t *testing.T) {
		got, _ := applyUpdate(baseRule(), UpdateInput{Status: str(StatusActive)}, today)
		if got.NextIndex != 3 || got.Status != StatusActive {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("an end date before the next occurrence finishes the rule", func(t *testing.T) {
		got, _ := applyUpdate(baseRule(), UpdateInput{EndDate: date(d(2026, 4, 1))}, today)
		if got.Status != StatusFinished {
			t.Errorf("status = %s, want finished", got.Status)
		}
	})

	t.Run("an end date after the next occurrence keeps it active", func(t *testing.T) {
		got, _ := applyUpdate(baseRule(), UpdateInput{EndDate: date(d(2026, 12, 31))}, today)
		if got.Status != StatusActive || got.EndDate == nil {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("end date before the start date is rejected", func(t *testing.T) {
		if _, err := applyUpdate(baseRule(), UpdateInput{EndDate: date(d(2025, 12, 31))}, today); !errors.Is(err, ErrEndBeforeStart) {
			t.Errorf("err = %v, want ErrEndBeforeStart", err)
		}
	})

	t.Run("a finished rule can't be changed", func(t *testing.T) {
		done := baseRule()
		done.Status = StatusFinished
		if _, err := applyUpdate(done, UpdateInput{Amount: i64(1)}, today); !errors.Is(err, ErrFinished) {
			t.Errorf("err = %v, want ErrFinished", err)
		}
	})
}

// fake repo to test the service's validation without a database
type fakeRepo struct{ created *NewRule }

func (f *fakeRepo) Create(_ context.Context, _ uuid.UUID, in NewRule) (Rule, error) {
	f.created = &in
	return Rule{StartDate: in.StartDate, NextRunOn: in.StartDate}, nil
}
func (f *fakeRepo) List(context.Context, uuid.UUID) ([]Rule, error) { return nil, nil }
func (f *fakeRepo) Update(context.Context, uuid.UUID, uuid.UUID, func(Rule) (Rule, error)) (Rule, error) {
	return Rule{}, nil
}
func (f *fakeRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func TestService_Create_Validation(t *testing.T){
	now := func() time.Time { return time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC) }
	good := CreateInput{
		Type: transaction.TypeExpense, Amount: 15_000_000, Frequency: FreqMonthly,
		StartDate: d(2026, 11, 1),
	}
	date := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name    string
		mutate  func(*CreateInput)
		wantErr error
	}{
		{"valid", func(*CreateInput) {}, nil},
		{"start exactly a year ago is allowed", func(in *CreateInput) { in.StartDate = d(2025, 10, 20) }, nil},
		{"start more than a year ago", func(in *CreateInput) { in.StartDate = d(2025, 10, 19) }, ErrStartTooOld},
		{"start far in the future is fine", func(in *CreateInput) { in.StartDate = d(2030, 1, 1) }, nil},
		{"end before start", func(in *CreateInput) { in.EndDate = date(d(2026, 10, 31)) }, ErrEndBeforeStart},
		{"end on the start date is fine", func(in *CreateInput) { in.EndDate = date(d(2026, 11, 1)) }, nil},
		{"bad frequency", func(in *CreateInput) { in.Frequency = "hourly" }, ErrInvalidFrequency},
		{"bad type", func(in *CreateInput) { in.Type = "transfer" }, transaction.ErrInvalidType},
		{"zero amount", func(in *CreateInput) { in.Amount = 0 }, transaction.ErrInvalidAmount},
		{"amount over the cap", func(in *CreateInput) { in.Amount = transaction.MaxAmount + 1 }, transaction.ErrInvalidAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeRepo{}, now)
			in := good
			tt.mutate(&in)
			_, err := svc.Create(context.Background(), uuid.New(), in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
