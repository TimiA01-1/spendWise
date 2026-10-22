package recurring

import (
	"time"
)

// occurrence returns the date of the n-th occurence of a rule (n = 0 is the start date itself)
func Occurrence(start time.Time, freq string, n int) time.Time {
	y, m, d := start.UTC().Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	switch freq {
	case FreqDaily:
		return start.AddDate(0, 0, n)
	case FreqWeekly:
		return start.AddDate(0, 0, 7*n)
	case FreqMonthly:
		return addMonths(start, n)
	case FreqYearly:
		return addMonths(start, 12*n)
	}
	return start // not reachable: frequencies are validated before a rule is saved
}


// addMonths moves forward n months, keeping the day of the month but clamping it to the last day of shorter months
 func addMonths(start time.Time, n int) time.Time {
	y, m, d := start.Date()
	total := int(m) - 1 + n
	year := y + total/12
	month := time.Month(total%12 + 1)

	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > lastDay {
		d = lastDay
	}
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

// maxSkip stops a runaway loop if someone resumes a daily rule after decades.
const maxSkip = 100_000

// firstIndexOnOrAfter returns the first occurence index >= from whose date is on or after day.
func firstIndexOnOrAfter(start time.Time, freq string, from int, day time.Time) int {
	n := from
	for i := 0; i < maxSkip && Occurrence(start, freq, n).Before(day); i++ {
		n++
	}
	return n
}