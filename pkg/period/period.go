package period

import (
	"errors"
	"time"
)

var ErrInvalidMonth = errors.New("month must look like 2026-10")

const monthLayout = "2006-01"

func Month(month string, now time.Time) (from, to time.Time, label string, err error) {
	if month == "" {
		n := now.UTC()
		from = time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		from, err = time.Parse(monthLayout, month)
		if err != nil {
			return time.Time{}, time.Time{}, "", ErrInvalidMonth
		}
	}
	return from, from.AddDate(0, 1, 0), from.Format(monthLayout), nil
}