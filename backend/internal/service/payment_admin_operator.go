package service

import "strings"

const fallbackPaymentAdminOperator = "admin"

func normalizePaymentAdminOperator(operator string) string {
	operator = strings.TrimSpace(operator)
	if operator == "" {
		return fallbackPaymentAdminOperator
	}
	return operator
}

func (p *RefundPlan) auditOperator() string {
	if p == nil {
		return fallbackPaymentAdminOperator
	}
	return normalizePaymentAdminOperator(p.AdminOperator)
}
