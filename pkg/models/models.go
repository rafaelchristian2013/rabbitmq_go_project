package models

type PaymentEvent struct {
	UserID        int     `json:"user_id"`
	DepositAmount float64 `json:"deposit_amount"`
}
