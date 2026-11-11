package orders

import (
	"testing"
	"time"
)

func TestReservationBoundary(t *testing.T) {
	expires := time.Date(2026, 11, 11, 12, 5, 0, 0, time.UTC)
	cases := []struct {
		name            string
		status          string
		now             time.Time
		canPay, expires bool
	}{
		{"before deadline", PendingPayment, expires.Add(-time.Nanosecond), true, false},
		{"exact deadline", PendingPayment, expires, false, true},
		{"after deadline", PendingPayment, expires.Add(time.Nanosecond), false, true},
		{"paid is terminal", Paid, expires.Add(time.Hour), false, false},
		{"expired is terminal", Expired, expires.Add(-time.Hour), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanPay(tc.status, tc.now, expires); got != tc.canPay {
				t.Fatalf("CanPay()=%v, want %v", got, tc.canPay)
			}
			if got := ShouldExpire(tc.status, tc.now, expires); got != tc.expires {
				t.Fatalf("ShouldExpire()=%v, want %v", got, tc.expires)
			}
		})
	}
}
