package orders

import "time"

const (
	PendingPayment = "PENDING_PAYMENT"
	Paid           = "PAID"
	Expired        = "EXPIRED"
)

func CanPay(status string, now, expiresAt time.Time) bool {
	return status == PendingPayment && now.Before(expiresAt)
}

func ShouldExpire(status string, now, expiresAt time.Time) bool {
	return status == PendingPayment && !now.Before(expiresAt)
}
