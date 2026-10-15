package alert

import (
	"strings"
	"testing"

	"spendWise/internal/budget"
)

func TestMessage(t *testing.T) {
	w := message("Food", budget.StatusWarning, 80)
	if !strings.Contains(w, "Food") || !strings.Contains(w, "80%") {
		t.Errorf("warning message = %q", w)
	}
	l := message("All spending", budget.StatusLimitReached, 104)
	if !strings.Contains(l, "All spending") || !strings.Contains(l, "limit") || !strings.Contains(l, "104%") {
		t.Errorf("limit message = %q", l)
	}
}