package admin

import (
	"net/http"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles admin payment management.
type PaymentHandler struct {
	paymentService *service.PaymentService
	configService  *service.PaymentConfigService
}

// NewPaymentHandler creates a new admin PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		configService:  configService,
	}
}

// --- Dashboard ---

// GetDashboard returns payment dashboard statistics.
// GET /api/v1/admin/payment/dashboard
func (h *PaymentHandler) GetDashboard(c *gin.Context) {
	h.GetDashboardGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) GetDashboardGateway(c gatewayctx.GatewayContext) {
	days := 30
	if d := c.QueryValue("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}
	stats, err := h.paymentService.GetDashboardStats(c.Request().Context(), days)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, stats)
}

// --- Orders ---

// ListOrders returns a paginated list of all payment orders.
// GET /api/v1/admin/payment/orders
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	h.ListOrdersGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) ListOrdersGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	var userID int64
	if uid := c.QueryValue("user_id"); uid != "" {
		if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
			userID = v
		}
	}
	orders, total, err := h.paymentService.AdminListOrders(c.Request().Context(), userID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.QueryValue("status"),
		OrderType:   c.QueryValue("order_type"),
		PaymentType: c.QueryValue("payment_type"),
		Keyword:     c.QueryValue("keyword"),
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, sanitizeAdminPaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

// GetOrderDetail returns detailed information about a single order.
// GET /api/v1/admin/payment/orders/:id
func (h *PaymentHandler) GetOrderDetail(c *gin.Context) {
	h.GetOrderDetailGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) GetOrderDetailGateway(c gatewayctx.GatewayContext) {
	orderID, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	order, err := h.paymentService.GetOrderByID(c.Request().Context(), orderID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	auditLogs, _ := h.paymentService.GetOrderAuditLogs(c.Request().Context(), orderID)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"order": sanitizeAdminPaymentOrderForResponse(order), "auditLogs": auditLogs})
}

// CancelOrder cancels a pending order (admin).
// POST /api/v1/admin/payment/orders/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	h.CancelOrderGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) CancelOrderGateway(c gatewayctx.GatewayContext) {
	orderID, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	operator := adminAuditOperatorFromGateway(c)
	msg, err := h.paymentService.AdminCancelOrder(c.Request().Context(), orderID, operator)
	if err != nil {
		logAdminAudit("payment", "cancel_order failed operator=%s order_id=%d error_reason=%s", operator, orderID, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("payment", "cancel_order succeeded operator=%s order_id=%d result=%s", operator, orderID, msg)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": msg})
}

// RetryFulfillment retries fulfillment for a paid order.
// POST /api/v1/admin/payment/orders/:id/retry
func (h *PaymentHandler) RetryFulfillment(c *gin.Context) {
	h.RetryFulfillmentGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) RetryFulfillmentGateway(c gatewayctx.GatewayContext) {
	orderID, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	operator := adminAuditOperatorFromGateway(c)
	if err := h.paymentService.RetryFulfillment(c.Request().Context(), orderID, operator); err != nil {
		logAdminAudit("payment", "retry_fulfillment failed operator=%s order_id=%d error_reason=%s", operator, orderID, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("payment", "retry_fulfillment succeeded operator=%s order_id=%d", operator, orderID)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "fulfillment retried"})
}

type adminPaymentOrderResponse struct {
	*dbent.PaymentOrder
	Currency string `json:"currency"`
}

func sanitizeAdminPaymentOrdersForResponse(orders []*dbent.PaymentOrder) []*adminPaymentOrderResponse {
	if len(orders) == 0 {
		return []*adminPaymentOrderResponse{}
	}
	out := make([]*adminPaymentOrderResponse, 0, len(orders))
	for _, order := range orders {
		out = append(out, sanitizeAdminPaymentOrderForResponse(order))
	}
	return out
}

func sanitizeAdminPaymentOrderForResponse(order *dbent.PaymentOrder) *adminPaymentOrderResponse {
	if order == nil {
		return nil
	}
	currency, _ := order.ProviderSnapshot["currency"].(string)
	cloned := *order
	cloned.ProviderSnapshot = nil
	return &adminPaymentOrderResponse{
		PaymentOrder: &cloned,
		Currency:     currency,
	}
}

// AdminProcessRefundRequest is the request body for admin refund processing.
type AdminProcessRefundRequest struct {
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Force         bool    `json:"force"`
	DeductBalance bool    `json:"deduct_balance"`
}

// ProcessRefund processes a refund for an order (admin).
// POST /api/v1/admin/payment/orders/:id/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	h.ProcessRefundGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) ProcessRefundGateway(c gatewayctx.GatewayContext) {
	orderID, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}

	var req AdminProcessRefundRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	operator := adminAuditOperatorFromGateway(c)
	plan, earlyResult, err := h.paymentService.PrepareRefund(c.Request().Context(), orderID, req.Amount, req.Reason, req.Force, req.DeductBalance)
	if err != nil {
		logAdminAudit("payment", "refund_prepare failed operator=%s order_id=%d amount=%.2f force=%t deduct_balance=%t error_reason=%s", operator, orderID, req.Amount, req.Force, req.DeductBalance, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if earlyResult != nil {
		logAdminAudit("payment", "refund_prepare returned operator=%s order_id=%d amount=%.2f require_force=%t", operator, orderID, req.Amount, earlyResult.RequireForce)
		response.SuccessContext(gatewayJSONResponder{ctx: c}, earlyResult)
		return
	}

	plan.AdminOperator = operator
	result, err := h.paymentService.ExecuteRefund(c.Request().Context(), plan)
	if err != nil {
		logAdminAudit("payment", "refund failed operator=%s order_id=%d amount=%.2f force=%t deduct_balance=%t error_reason=%s", operator, orderID, req.Amount, req.Force, req.DeductBalance, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("payment", "refund succeeded operator=%s order_id=%d amount=%.2f force=%t deduct_balance=%t", operator, orderID, req.Amount, req.Force, req.DeductBalance)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// --- Subscription Plans ---

// ListPlans returns all subscription plans.
// GET /api/v1/admin/payment/plans
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	h.ListPlansGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) ListPlansGateway(c gatewayctx.GatewayContext) {
	plans, err := h.configService.ListPlans(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	groupInfo := h.configService.GetGroupInfoMap(c.Request().Context(), plans)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, adminSubscriptionPlansForResponse(plans, groupInfo))
}

type AdminSubscriptionPlanResult struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"`
	GroupPlatform   string    `json:"group_platform,omitempty"`
	GroupName       string    `json:"group_name,omitempty"`
	RateMultiplier  float64   `json:"rate_multiplier,omitempty"`
	DailyLimitUSD   *float64  `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD  *float64  `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD *float64  `json:"monthly_limit_usd,omitempty"`
	ModelScopes     []string  `json:"supported_model_scopes,omitempty"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Price           float64   `json:"price"`
	OriginalPrice   *float64  `json:"original_price,omitempty"`
	Currency        string    `json:"currency,omitempty"`
	ValidityDays    int       `json:"validity_days"`
	ValidityUnit    string    `json:"validity_unit"`
	Features        string    `json:"features"`
	ProductName     string    `json:"product_name"`
	ForSale         bool      `json:"for_sale"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

func adminSubscriptionPlansForResponse(plans []*dbent.SubscriptionPlan, groupInfo map[int64]service.PlanGroupInfo) []AdminSubscriptionPlanResult {
	result := make([]AdminSubscriptionPlanResult, 0, len(plans))
	for _, p := range plans {
		if p == nil {
			continue
		}
		gi := groupInfo[p.GroupID]
		result = append(result, AdminSubscriptionPlanResult{
			ID:              int64(p.ID),
			GroupID:         p.GroupID,
			GroupPlatform:   gi.Platform,
			GroupName:       gi.Name,
			RateMultiplier:  gi.RateMultiplier,
			DailyLimitUSD:   gi.DailyLimitUSD,
			WeeklyLimitUSD:  gi.WeeklyLimitUSD,
			MonthlyLimitUSD: gi.MonthlyLimitUSD,
			ModelScopes:     gi.ModelScopes,
			Name:            p.Name,
			Description:     p.Description,
			Price:           p.Price,
			OriginalPrice:   p.OriginalPrice,
			Currency:        p.Currency,
			ValidityDays:    p.ValidityDays,
			ValidityUnit:    p.ValidityUnit,
			Features:        p.Features,
			ProductName:     p.ProductName,
			ForSale:         p.ForSale,
			SortOrder:       p.SortOrder,
			CreatedAt:       p.CreatedAt,
			UpdatedAt:       p.UpdatedAt,
		})
	}
	return result
}

// CreatePlan creates a new subscription plan.
// POST /api/v1/admin/payment/plans
func (h *PaymentHandler) CreatePlan(c *gin.Context) {
	h.CreatePlanGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) CreatePlanGateway(c gatewayctx.GatewayContext) {
	var req service.CreatePlanRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.CreatePlan(c.Request().Context(), req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.CreatedContext(gatewayJSONResponder{ctx: c}, plan)
}

// UpdatePlan updates an existing subscription plan.
// PUT /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) UpdatePlan(c *gin.Context) {
	h.UpdatePlanGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) UpdatePlanGateway(c gatewayctx.GatewayContext) {
	id, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	var req service.UpdatePlanRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.UpdatePlan(c.Request().Context(), id, req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, plan)
}

// DeletePlan deletes a subscription plan.
// DELETE /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) DeletePlan(c *gin.Context) {
	h.DeletePlanGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) DeletePlanGateway(c gatewayctx.GatewayContext) {
	id, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeletePlan(c.Request().Context(), id); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "deleted"})
}

// --- Provider Instances ---

// ListProviders returns all payment provider instances.
// GET /api/v1/admin/payment/providers
func (h *PaymentHandler) ListProviders(c *gin.Context) {
	h.ListProvidersGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) ListProvidersGateway(c gatewayctx.GatewayContext) {
	providers, err := h.configService.ListProviderInstancesWithConfig(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, providers)
}

// CreateProvider creates a new payment provider instance.
// POST /api/v1/admin/payment/providers
func (h *PaymentHandler) CreateProvider(c *gin.Context) {
	h.CreateProviderGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) CreateProviderGateway(c gatewayctx.GatewayContext) {
	var req service.CreateProviderInstanceRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.CreateProviderInstance(c.Request().Context(), req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request().Context())
	response.CreatedContext(gatewayJSONResponder{ctx: c}, inst)
}

// UpdateProvider updates an existing payment provider instance.
// PUT /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) UpdateProvider(c *gin.Context) {
	h.UpdateProviderGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) UpdateProviderGateway(c gatewayctx.GatewayContext) {
	id, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	var req service.UpdateProviderInstanceRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.UpdateProviderInstance(c.Request().Context(), id, req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request().Context())
	response.SuccessContext(gatewayJSONResponder{ctx: c}, inst)
}

// DeleteProvider deletes a payment provider instance.
// DELETE /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) DeleteProvider(c *gin.Context) {
	h.DeleteProviderGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) DeleteProviderGateway(c gatewayctx.GatewayContext) {
	id, ok := parseIDParamGateway(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeleteProviderInstance(c.Request().Context(), id); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request().Context())
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "deleted"})
}

// parseIDParam parses an int64 path parameter.
// Returns the parsed ID and true on success; on failure it writes a BadRequest response and returns false.
func parseIDParam(c *gin.Context, paramName string) (int64, bool) {
	return parseIDParamGateway(gatewayctx.FromGin(c), paramName)
}

func parseIDParamGateway(c gatewayctx.GatewayContext, paramName string) (int64, bool) {
	value := ""
	if c != nil {
		value = c.PathParam(paramName)
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid "+paramName)
		return 0, false
	}
	return id, true
}

// --- Config ---

// GetConfig returns the payment configuration (admin view).
// GET /api/v1/admin/payment/config
func (h *PaymentHandler) GetConfig(c *gin.Context) {
	h.GetConfigGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) GetConfigGateway(c gatewayctx.GatewayContext) {
	cfg, err := h.configService.GetPaymentConfig(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateConfig updates the payment configuration.
// PUT /api/v1/admin/payment/config
func (h *PaymentHandler) UpdateConfig(c *gin.Context) {
	h.UpdateConfigGateway(gatewayctx.FromGin(c))
}

func (h *PaymentHandler) UpdateConfigGateway(c gatewayctx.GatewayContext) {
	var req service.UpdatePaymentConfigRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdatePaymentConfig(c.Request().Context(), req); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "updated"})
}
