package admin

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"net/http"
)

func (h *SettingHandler) TestSMTPConnectionGateway(c gatewayctx.GatewayContext) {
	var req TestSMTPRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}

	password := req.SMTPPassword
	if password == "" {
		savedConfig, err := h.emailService.GetSMTPConfig(c.Request().Context())
		if err == nil && savedConfig != nil {
			password = savedConfig.Password
		}
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		UseTLS:   req.SMTPUseTLS,
	}

	err := h.emailService.TestSMTPConnectionWithConfig(config)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "SMTP connection test failed: "+err.Error())
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "SMTP connection successful"})
}

func (h *SettingHandler) SendTestEmailGateway(c gatewayctx.GatewayContext) {
	var req SendTestEmailRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}

	password := req.SMTPPassword
	if password == "" {
		savedConfig, err := h.emailService.GetSMTPConfig(c.Request().Context())
		if err == nil && savedConfig != nil {
			password = savedConfig.Password
		}
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		From:     req.SMTPFrom,
		FromName: req.SMTPFromName,
		UseTLS:   req.SMTPUseTLS,
	}

	siteName := h.settingService.GetSiteName(c.Request().Context())
	subject := "[" + siteName + "] Test Email"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
        .content { padding: 40px 30px; text-align: center; }
        .success { color: #10b981; font-size: 48px; margin-bottom: 20px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + siteName + `</h1>
        </div>
        <div class="content">
            <div class="success">✓</div>
            <h2>Email Configuration Successful!</h2>
            <p>This is a test email to verify your SMTP settings are working correctly.</p>
        </div>
        <div class="footer">
            <p>This is an automated test message.</p>
        </div>
    </div>
</body>
</html>
`

	if err := h.emailService.SendEmailWithConfig(config, req.Email, subject, body); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Failed to send test email: "+err.Error())
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "Test email sent successfully"})
}

func (h *SettingHandler) GetAdminAPIKeyGateway(c gatewayctx.GatewayContext) {
	maskedKey, exists, err := h.settingService.GetAdminAPIKeyStatus(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"exists":     exists,
		"masked_key": maskedKey,
	})
}

func (h *SettingHandler) RegenerateAdminAPIKeyGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Authorization required")
		return
	}
	operator := adminAuditOperatorFromGateway(c)

	adminTokenVersion := int64(0)
	if h.userService != nil {
		admin, err := h.userService.GetByID(c.Request().Context(), subject.UserID)
		if err != nil || admin == nil || !admin.IsActive() || !admin.IsAdmin() {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Admin user not found")
			return
		}
		adminTokenVersion = admin.TokenVersion
	}

	key, err := h.settingService.GenerateAdminAPIKey(c.Request().Context(), subject.UserID, adminTokenVersion)
	if err != nil {
		logAdminAudit("settings", "admin_api_key_regenerate failed operator=%s error_reason=%s", operator, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("settings", "admin_api_key_regenerate succeeded operator=%s", operator)

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"key": key,
	})
}

func (h *SettingHandler) DeleteAdminAPIKeyGateway(c gatewayctx.GatewayContext) {
	operator := adminAuditOperatorFromGateway(c)
	if err := h.settingService.DeleteAdminAPIKey(c.Request().Context()); err != nil {
		logAdminAudit("settings", "admin_api_key_delete failed operator=%s error_reason=%s", operator, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("settings", "admin_api_key_delete succeeded operator=%s", operator)

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "Admin API key deleted"})
}

func (h *SettingHandler) GetOverloadCooldownSettingsGateway(c gatewayctx.GatewayContext) {
	settings, err := h.settingService.GetOverloadCooldownSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.OverloadCooldownSettings{
		Enabled:         settings.Enabled,
		CooldownMinutes: settings.CooldownMinutes,
	})
}

func (h *SettingHandler) UpdateOverloadCooldownSettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateOverloadCooldownSettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	settings := &service.OverloadCooldownSettings{
		Enabled:         req.Enabled,
		CooldownMinutes: req.CooldownMinutes,
	}

	if err := h.settingService.SetOverloadCooldownSettings(c.Request().Context(), settings); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	updatedSettings, err := h.settingService.GetOverloadCooldownSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.OverloadCooldownSettings{
		Enabled:         updatedSettings.Enabled,
		CooldownMinutes: updatedSettings.CooldownMinutes,
	})
}

func (h *SettingHandler) GetStreamTimeoutSettingsGateway(c gatewayctx.GatewayContext) {
	settings, err := h.settingService.GetStreamTimeoutSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.StreamTimeoutSettings{
		Enabled:                settings.Enabled,
		Action:                 settings.Action,
		TempUnschedMinutes:     settings.TempUnschedMinutes,
		ThresholdCount:         settings.ThresholdCount,
		ThresholdWindowMinutes: settings.ThresholdWindowMinutes,
	})
}

func (h *SettingHandler) GetRectifierSettingsGateway(c gatewayctx.GatewayContext) {
	settings, err := h.settingService.GetRectifierSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.RectifierSettings{
		Enabled:                  settings.Enabled,
		ThinkingSignatureEnabled: settings.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    settings.ThinkingBudgetEnabled,
	})
}

func (h *SettingHandler) UpdateRectifierSettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateRectifierSettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	settings := &service.RectifierSettings{
		Enabled:                  req.Enabled,
		ThinkingSignatureEnabled: req.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    req.ThinkingBudgetEnabled,
	}

	if err := h.settingService.SetRectifierSettings(c.Request().Context(), settings); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	updatedSettings, err := h.settingService.GetRectifierSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.RectifierSettings{
		Enabled:                  updatedSettings.Enabled,
		ThinkingSignatureEnabled: updatedSettings.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    updatedSettings.ThinkingBudgetEnabled,
	})
}

func (h *SettingHandler) GetBetaPolicySettingsGateway(c gatewayctx.GatewayContext) {
	settings, err := h.settingService.GetBetaPolicySettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	rules := make([]dto.BetaPolicyRule, len(settings.Rules))
	for i, r := range settings.Rules {
		rules[i] = dto.BetaPolicyRule(r)
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.BetaPolicySettings{Rules: rules})
}

func (h *SettingHandler) UpdateBetaPolicySettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateBetaPolicySettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	rules := make([]service.BetaPolicyRule, len(req.Rules))
	for i, r := range req.Rules {
		rules[i] = service.BetaPolicyRule(r)
	}

	settings := &service.BetaPolicySettings{Rules: rules}
	if err := h.settingService.SetBetaPolicySettings(c.Request().Context(), settings); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.settingService.GetBetaPolicySettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	outRules := make([]dto.BetaPolicyRule, len(updated.Rules))
	for i, r := range updated.Rules {
		outRules[i] = dto.BetaPolicyRule(r)
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.BetaPolicySettings{Rules: outRules})
}

func (h *SettingHandler) UpdateStreamTimeoutSettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateStreamTimeoutSettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	settings := &service.StreamTimeoutSettings{
		Enabled:                req.Enabled,
		Action:                 req.Action,
		TempUnschedMinutes:     req.TempUnschedMinutes,
		ThresholdCount:         req.ThresholdCount,
		ThresholdWindowMinutes: req.ThresholdWindowMinutes,
	}

	if err := h.settingService.SetStreamTimeoutSettings(c.Request().Context(), settings); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	updatedSettings, err := h.settingService.GetStreamTimeoutSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.StreamTimeoutSettings{
		Enabled:                updatedSettings.Enabled,
		Action:                 updatedSettings.Action,
		TempUnschedMinutes:     updatedSettings.TempUnschedMinutes,
		ThresholdCount:         updatedSettings.ThresholdCount,
		ThresholdWindowMinutes: updatedSettings.ThresholdWindowMinutes,
	})
}
