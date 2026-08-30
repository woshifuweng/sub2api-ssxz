package service

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidDashboardOperationsRange = errors.New("invalid dashboard operations time range")

type DashboardOperationsTopCustomer struct {
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Username   string  `json:"username"`
	ActualCost float64 `json:"actual_cost"`
	Requests   int64   `json:"requests"`
	ActiveKeys int64   `json:"active_keys"`
}

type DashboardOperationsSummary struct {
	StartDate             time.Time                        `json:"start_date"`
	EndDate               time.Time                        `json:"end_date"`
	NewCustomers          int64                            `json:"new_customers"`
	CustomerActualCost    float64                          `json:"customer_actual_cost"`
	InviteeRechargeAmount float64                          `json:"invitee_recharge_amount"`
	RebatePending         float64                          `json:"rebate_pending"`
	RebateAvailable       float64                          `json:"rebate_available"`
	RebateTransferred     float64                          `json:"rebate_transferred"`
	ActiveCustomers       int64                            `json:"active_customers"`
	ActiveAPIKeys         int64                            `json:"active_api_keys"`
	TopCustomers          []DashboardOperationsTopCustomer `json:"top_customers"`
}

type DashboardOperationsRepository interface {
	GetOperationsSummary(ctx context.Context, startTime, endTime time.Time, topLimit int) (*DashboardOperationsSummary, error)
}

type DashboardOperationsService struct {
	repo DashboardOperationsRepository
}

func NewDashboardOperationsService(repo DashboardOperationsRepository) *DashboardOperationsService {
	return &DashboardOperationsService{repo: repo}
}

func (s *DashboardOperationsService) GetSummary(ctx context.Context, startTime, endTime time.Time, topLimit int) (*DashboardOperationsSummary, error) {
	if !endTime.After(startTime) {
		return nil, ErrInvalidDashboardOperationsRange
	}
	if topLimit <= 0 {
		topLimit = 10
	}
	if topLimit > 50 {
		topLimit = 50
	}
	return s.repo.GetOperationsSummary(ctx, startTime, endTime, topLimit)
}
