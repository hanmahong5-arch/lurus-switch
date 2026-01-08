package biz

import (
	"context"
	"time"
)

// PaymentStatus defines payment status
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusSuccess  PaymentStatus = "success"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentMethod defines payment methods
type PaymentMethod string

const (
	PaymentMethodStripe PaymentMethod = "stripe"
	PaymentMethodAlipay PaymentMethod = "alipay"
	PaymentMethodWechat PaymentMethod = "wechat"
	PaymentMethodManual PaymentMethod = "manual"
)

// Payment represents a payment record
type Payment struct {
	ID             int64         `json:"id" gorm:"primaryKey"`
	UserID         int           `json:"user_id" gorm:"index;not null"`
	SubscriptionID int64         `json:"subscription_id" gorm:"index"`
	PlanID         int64         `json:"plan_id" gorm:"index"`
	AmountCents    int           `json:"amount_cents" gorm:"not null"`
	Currency       string        `json:"currency" gorm:"size:3;default:'CNY'"`
	Method         PaymentMethod `json:"method" gorm:"size:20"`
	Status         PaymentStatus `json:"status" gorm:"size:20;default:'pending'"`
	ExternalID     string        `json:"external_id,omitempty" gorm:"size:100;index"`
	Description    string        `json:"description,omitempty" gorm:"size:255"`
	Metadata       string        `json:"metadata,omitempty" gorm:"type:text"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	RefundedAt     *time.Time    `json:"refunded_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Payment) TableName() string {
	return "payments"
}

// PaymentRepo defines the payment repository interface
type PaymentRepo interface {
	Create(ctx context.Context, payment *Payment) error
	Update(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id int64) (*Payment, error)
	GetByExternalID(ctx context.Context, externalID string) (*Payment, error)
	ListByUserID(ctx context.Context, userID int, offset, limit int) ([]*Payment, int64, error)
	ListBySubscriptionID(ctx context.Context, subID int64) ([]*Payment, error)
}

// PaymentUsecase defines the payment business logic
type PaymentUsecase struct {
	paymentRepo PaymentRepo
	subUsecase  *SubscriptionUsecase
}

// NewPaymentUsecase creates a new payment usecase
func NewPaymentUsecase(paymentRepo PaymentRepo, subUsecase *SubscriptionUsecase) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo: paymentRepo,
		subUsecase:  subUsecase,
	}
}

// CreatePayment creates a new payment record
func (uc *PaymentUsecase) CreatePayment(ctx context.Context, userID int, planID int64, amountCents int, method PaymentMethod) (*Payment, error) {
	payment := &Payment{
		UserID:      userID,
		PlanID:      planID,
		AmountCents: amountCents,
		Method:      method,
		Status:      PaymentStatusPending,
	}

	if err := uc.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// ConfirmPayment confirms a payment and activates the subscription
func (uc *PaymentUsecase) ConfirmPayment(ctx context.Context, paymentID int64, externalID string) error {
	payment, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	now := time.Now()
	payment.Status = PaymentStatusSuccess
	payment.ExternalID = externalID
	payment.PaidAt = &now

	if err := uc.paymentRepo.Update(ctx, payment); err != nil {
		return err
	}

	// Get plan code and create subscription
	// This is simplified - in production, would need to look up plan
	return nil
}

// GetUserPayments returns payment history for a user
func (uc *PaymentUsecase) GetUserPayments(ctx context.Context, userID int, page, pageSize int) ([]*Payment, int64, error) {
	offset := (page - 1) * pageSize
	return uc.paymentRepo.ListByUserID(ctx, userID, offset, pageSize)
}
