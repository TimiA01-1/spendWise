package recurring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"spendWise/internal/transaction"
)

const(
	maxNoteLen = 500
	maxBackfill = 365 * 24 * time.Hour
)
var (
	ErrInvalidFrequency = errors.New("frequency must be daily, weekly, monthly or yearly")
	ErrInvalidStatus = errors.New("status must be active or paused")
	ErrStartTooOld = errors.New("start_date cannot be more than a year in the past")
	ErrEndBeforeStart = errors.New("end_date must not be before start_date")
	ErrFinished = errors.New("this rule has finished and can no longer be changed")
)

type Service struct {
	repo Repository
	now  func() time.Time // injected so tests (and the sandbox) can control "today"
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

type CreateInput struct {
	AccountID  uuid.UUID
	CategoryID uuid.UUID
	Type string
	Amount int64
	Note string
	Frequency  string
	StartDate  time.Time
	EndDate *time.Time
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (Rule, error){
	if in.Type != transaction.TypeIncome && in.Type != transaction.TypeExpense {
		return Rule{}, transaction.ErrInvalidType
	}
	if in.Amount <= 0 || in.Amount > transaction.MaxAmount {
		return Rule{}, transaction.ErrInvalidAmount
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > maxNoteLen {
		return Rule{}, transaction.ErrNoteTooLong
	}
	if !validFrequency(in.Frequency) {
		return Rule{}, ErrInvalidFrequency
	}

	start := dateOnly(in.StartDate)
	if start.Before(s.today().Add(-maxBackfill)) {
		return Rule{}, ErrStartTooOld
	}
	var end *time.Time
	if in.EndDate != nil {
		e := dateOnly(*in.EndDate)
		if e.Before(start) {
			return Rule{}, ErrEndBeforeStart
		}
		end = &e
	}
	rule, err := s.repo.Create(ctx, userID, NewRule{
		AccountID:  in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		Amount: in.Amount,
		Note: note,
		Frequency: in.Frequency,
		StartDate: start,
		EndDate: end,
	})
	if err != nil {
		return Rule{}, passThrough("create recurring rule", err)
	}
	return rule, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID)([]Rule, error){
	rules, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list recurring rules: %w", err)
	}
	return rules, nil
}

// UpdateInput uses pointers to tell "not sent" (nil) from "sent"
type UpdateInput struct {
	Amount  *int64
	Note    *string
	EndDate *time.Time
	Status  *string // "active" or "paused"
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput)(Rule, error){
	if in.Amount != nil && (*in.Amount <= 0 || *in.Amount > transaction.MaxAmount) {
		return Rule{}, transaction.ErrInvalidAmount
	}
	if in.Note != nil && utf8.RuneCountInString(strings.TrimSpace(*in.Note)) > maxNoteLen {
		return Rule{}, transaction.ErrNoteTooLong
	}
	if in.Status != nil && *in.Status != StatusActive && *in.Status != StatusPaused {
		return Rule{}, ErrInvalidStatus
	}

	today := s.today()
	rule, err := s.repo.Update(ctx, userID, id, func(current Rule) (Rule, error) {
		return applyUpdate(current, in, today)
	})
	if err != nil {
		return Rule{}, passThrough("update recurring rule", err)
	}
	return rule, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error{
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return passThrough("delete recurring rule", err)
	}
	return nil
}

// applyUpdates computes the edited rule
func applyUpdate(cur Rule, in UpdateInput, today time.Time)(Rule, error){
	if cur.Status == StatusFinished {
		return Rule{}, ErrFinished
	}
	r := cur

	if in.Amount != nil {
		r.Amount = *in.Amount
	}
	if in.Note != nil {
		r.Note = strings.TrimSpace(*in.Note)
	}
	if in.EndDate != nil {
		e := dateOnly(*in.EndDate)
		if e.Before(r.StartDate) {
			return Rule{}, ErrEndBeforeStart
		}
		r.EndDate = &e
	}
	if in.Status != nil {
		switch {
		case *in.Status == StatusPaused:
			r.Status = StatusPaused
		case r.Status == StatusPaused: // active requested, and it was paused: resume
			r = resume(r, today)
		}
	}
	// if the new end date is before the next occurence, nothing is left to run
	if r.EndDate != nil && r.NextRunOn.After(*r.EndDate){
		r.Status = StatusFinished
	}
	return r, nil
}

// resume reactivates a paused rule
func resume(r Rule, today time.Time) Rule {
	r.Status = StatusActive
	r.NextIndex = firstIndexOnOrAfter(r.StartDate, r.Frequency, r.NextIndex, today)
	r.NextRunOn = Occurrence(r.StartDate, r.Frequency, r.NextIndex)
	return r
}

func (s *Service) today() time.Time { return dateOnly(s.now()) }

func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// passThrough keeps domain errors unchanged
func passThrough(op string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, ErrFinished),
		errors.Is(err, ErrEndBeforeStart),
		errors.Is(err, transaction.ErrAccountNotFound),
		errors.Is(err, transaction.ErrCategoryNotFound),
		errors.Is(err, transaction.ErrCategoryTypeMismatch):
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
