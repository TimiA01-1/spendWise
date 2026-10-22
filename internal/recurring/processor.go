package recurring

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/db"
	"spendWise/internal/transaction"
)

//maxPerRun caps how many occurences one pass handles, so a backlog wont keep the worker busy forever
const maxPerRun = 1000

// processor is the background worker that turns due rules into transactions
type Processor struct{
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
}

func NewProcessor(pool *pgxpool.Pool, q *db.Queries, now func() time.Time) *Processor {
	return &Processor{pool: pool, q: q, now: now}
}

// run processes due rules immediately at every interval until ctx is cancelled
func (p *Processor) Run(ctx context.Context, every time.Duration) {
	log.Printf("recurring: worker started (checking every %s)", every)
	p.tick(ctx)

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("recurring: worker stopped")
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Processor) tick(ctx context.Context) {
	n, err := p.RunOnce(ctx)
	if n > 0 {
		log.Printf("recurring: created %d transaction(s)", n)
	}
	if err != nil && ctx.Err() == nil {
		log.Printf("recurring: run failed: %v", err)
	}
}

type outcome int

const (
	outcomeNoneDue outcome = iota
	outcomeCreated
	outcomeRuleFailed
)


// RunOnce creates every transaction that is due as of today and returns how many it created
func (p *Processor) RunOnce(ctx context.Context)(int, error){
	today := dateOnly(p.now())
	created := 0

	for attempts := 0; attempts < maxPerRun; attempts++ {
		if err := ctx.Err(); err != nil {
			return created, err
		}
		res, err := p.processOne(ctx, today)
		if err != nil {
			return created, err
		}
		switch res {
		case outcomeNoneDue:
			return created, nil
		case outcomeCreated:
			created++
		}
	}
	return created, nil
}

// processOne claims one due rule and creates its next occurrence.
func (p *Processor) processOne(ctx context.Context, today time.Time) (outcome, error){
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return outcomeNoneDue, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // undoes everything unless we commit
	qtx := p.q.WithTx(tx)

	rule, err := qtx.ClaimDueRecurring(ctx, today)
	if errors.Is(err, pgx.ErrNoRows) {
		return outcomeNoneDue, nil // nothing (unlocked) is due
	}
	if err != nil {
		return outcomeNoneDue, fmt.Errorf("claim rule: %w", err)
	}

	err = p.createOccurrence(ctx, qtx, rule)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return outcomeNoneDue, fmt.Errorf("commit: %w", err)
		}
		return outcomeCreated, nil
	}

	if isRuleProblem(err) {
		_ = tx.Rollback(ctx) // release the lock before recording the failure
		if ferr := p.q.FailRecurring(ctx, db.FailRecurringParams{ID: rule.ID, LastError: err.Error()}); ferr != nil {
			return outcomeNoneDue, fmt.Errorf("record failure: %w", ferr)
		}
		log.Printf("recurring: rule %s failed: %v", rule.ID, err)
		return outcomeRuleFailed, nil
	}

	return outcomeNoneDue, err
}

func (p *Processor) createOccurrence(ctx context.Context, qtx *db.Queries, rule db.ClaimDueRecurringRow)error{
	t, err := transaction.RecordInTx(ctx, qtx, rule.UserID, transaction.NewTransaction{
		AccountID:  rule.AccountID,
		CategoryID: rule.CategoryID,
		Type: rule.Type,
		Amount: rule.Amount,
		Note: rule.Note,
		Date: rule.NextRunOn,
	})
	if err != nil {
		return err
	}

	if err := qtx.LinkTransactionToRecurring(ctx, db.LinkTransactionToRecurringParams{
		ID:           t.ID,
		RecurringID:  rule.ID,
		RecurringFor: rule.NextRunOn,
	}); err != nil {
		return fmt.Errorf("link transaction to rule: %w", err)
	}

	nextIndex := int(rule.NextIndex) + 1
	next := Occurrence(rule.StartDate, rule.Frequency, nextIndex)
	status := StatusActive
	if !rule.EndDate.IsZero() && next.After(rule.EndDate) {
		status = StatusFinished
	}
	if err := qtx.AdvanceRecurring(ctx, db.AdvanceRecurringParams{
		ID:        rule.ID,
		NextIndex: int32(nextIndex),
		NextRunOn: next,
		Status:    status,
	}); err != nil {
		return fmt.Errorf("advance rule: %w", err)
	}
	return nil
}

// isRuleProblem reports if err means "this rule is no longer valid" as opposed to a temporary infrastructure problem
func isRuleProblem(err error) bool {
	return errors.Is(err, transaction.ErrAccountNotFound) ||
		errors.Is(err, transaction.ErrCategoryNotFound) ||
		errors.Is(err, transaction.ErrCategoryTypeMismatch)
}