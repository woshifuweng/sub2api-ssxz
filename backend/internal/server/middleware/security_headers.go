package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

const (
	// CSPNonceKey is the context key for storing the CSP nonce
	CSPNonceKey = "csp_nonce"
	// NonceTemplate is the placeholder in CSP policy for nonce
	NonceTemplate = "__CSP_NONCE__"
	// CloudflareInsightsDomain is the domain for Cloudflare Web Analytics
	CloudflareInsightsDomain = "https://static.cloudflareinsights.com"
	// StripeJSDomain is required by @stripe/stripe-js.
	StripeJSDomain = "https://js.stripe.com"
	// StripeHooksDomain is used by Stripe hosted payment frames.
	StripeHooksDomain = "https://hooks.stripe.com"
	// StripeAPIDomain is used by Stripe Elements.
	StripeAPIDomain = "https://api.stripe.com"
	// StripeQDomain is used by Stripe telemetry.
	StripeQDomain = "https://q.stripe.com"
	// StripeRDomain is used by Stripe telemetry.
	StripeRDomain = "https://r.stripe.com"
	// ChainDianShopDomain is the hosted recharge shop embedded in the user page.
	ChainDianShopDomain = "https://pay.ldxp.cn"
	// HSTSHeaderValue enables one year of HTTPS-only access in production.
	HSTSHeaderValue = "max-age=31536000"
)

// GenerateNonce generates a cryptographically secure random nonce.
// 返回 error 以确保调用方在 crypto/rand 失败时能正确降级。
func GenerateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate CSP nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GetNonceFromContext retrieves the CSP nonce from gin context
func GetNonceFromContext(c *gin.Context) string {
	return GetNonceFromGatewayContext(gatewayctx.FromGin(c))
}

func GetNonceFromGatewayContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	if nonce, exists := c.Value(CSPNonceKey); exists {
		if s, ok := nonce.(string); ok {
			return s
		}
	}
	return ""
}

// SecurityHeaders sets baseline security headers for all responses.
// getFrameSrcOrigins is an optional function that returns extra origins to inject into frame-src;
// pass nil to disable dynamic frame-src injection.
func SecurityHeaders(cfg config.CSPConfig, getFrameSrcOrigins func() []string) gin.HandlerFunc {
	policy := prepareSecurityHeadersPolicy(cfg)

	return func(c *gin.Context) {
		ApplySecurityHeadersContext(gatewayctx.FromGin(c), cfg, policy, getFrameSrcOrigins)
		c.Next()
	}
}

func prepareSecurityHeadersPolicy(cfg config.CSPConfig) string {
	policy := strings.TrimSpace(cfg.Policy)
	if policy == "" {
		policy = config.DefaultCSPPolicy
	}
	return enhanceCSPPolicy(policy)
}

func ApplySecurityHeadersContext(c gatewayctx.GatewayContext, cfg config.CSPConfig, preparedPolicy string, getFrameSrcOrigins func() []string) {
	if c == nil {
		return
	}
	finalPolicy := preparedPolicy
	if strings.TrimSpace(finalPolicy) == "" {
		finalPolicy = prepareSecurityHeadersPolicy(cfg)
	}
	if getFrameSrcOrigins != nil {
		for _, origin := range getFrameSrcOrigins() {
			if origin != "" {
				finalPolicy = addToDirective(finalPolicy, "frame-src", origin)
			}
		}
	}

	c.SetHeader("X-Content-Type-Options", "nosniff")
	// /image/* is intentionally embedded as an iframe inside the main app;
	// allow same-origin framing while still blocking cross-origin embedding.
	if isEmbeddablePath(c) {
		c.SetHeader("X-Frame-Options", "SAMEORIGIN")
	} else {
		c.SetHeader("X-Frame-Options", "DENY")
	}
	c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
	if gin.Mode() == gin.ReleaseMode {
		c.SetHeader("Strict-Transport-Security", HSTSHeaderValue)
	}
	if isAPIRoutePathContext(c) {
		return
	}

	if cfg.Enabled {
		nonce, err := GenerateNonce()
		if err != nil {
			log.Printf("[SecurityHeaders] %v — 降级为无 nonce 的 CSP", err)
			policy := strings.ReplaceAll(finalPolicy, NonceTemplate, "'unsafe-inline'")
			if isEmbeddablePath(c) {
				policy = strings.ReplaceAll(policy, "frame-ancestors 'none'", "frame-ancestors 'self'")
			}
			c.SetHeader("Content-Security-Policy", policy)
			return
		}
		c.SetValue(CSPNonceKey, nonce)
		policy := strings.ReplaceAll(finalPolicy, NonceTemplate, "'nonce-"+nonce+"'")
		if isEmbeddablePath(c) {
			policy = strings.ReplaceAll(policy, "frame-ancestors 'none'", "frame-ancestors 'self'")
		}
		c.SetHeader("Content-Security-Policy", policy)
	}
}

// isEmbeddablePath returns true for paths that are intentionally served as
// same-origin iframes within the main application (e.g. the image workbench).
func isEmbeddablePath(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	path := c.Path()
	return path == "/image" || strings.HasPrefix(path, "/image/")
}

func isAPIRoutePathContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	path := c.Path()
	return strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/v1beta/") ||
		strings.HasPrefix(path, "/antigravity/") ||
		strings.HasPrefix(path, "/sora/") ||
		strings.HasPrefix(path, "/responses")
}

func isAPIRoutePath(c *gin.Context) bool {
	return isAPIRoutePathContext(gatewayctx.FromGin(c))
}

// enhanceCSPPolicy ensures the CSP policy includes nonce support and required third-party domains.
// This allows the application to work correctly even if the config file has an older CSP policy.
func enhanceCSPPolicy(policy string) string {
	// Add nonce placeholder to script-src if not present
	if !strings.Contains(policy, NonceTemplate) && !strings.Contains(policy, "'nonce-") {
		policy = addToDirective(policy, "script-src", NonceTemplate)
	}

	// Add Cloudflare Insights domain to script-src if not present
	if !strings.Contains(policy, CloudflareInsightsDomain) {
		policy = addToDirective(policy, "script-src", CloudflareInsightsDomain)
	}
	if !strings.Contains(policy, StripeJSDomain) {
		policy = addToDirective(policy, "script-src", StripeJSDomain)
		policy = addToDirective(policy, "frame-src", StripeJSDomain)
	}
	if !strings.Contains(policy, StripeHooksDomain) {
		policy = addToDirective(policy, "frame-src", StripeHooksDomain)
	}
	if !strings.Contains(policy, StripeAPIDomain) {
		policy = addToDirective(policy, "connect-src", StripeAPIDomain)
	}
	if !strings.Contains(policy, StripeQDomain) {
		policy = addToDirective(policy, "connect-src", StripeQDomain)
	}
	if !strings.Contains(policy, StripeRDomain) {
		policy = addToDirective(policy, "connect-src", StripeRDomain)
	}
	if !strings.Contains(policy, ChainDianShopDomain) {
		policy = addToDirective(policy, "frame-src", ChainDianShopDomain)
	}

	// The image workbench (/image/) is same-origin and is embedded as an iframe
	// by /app/image. An explicitly listed frame-src does NOT inherit from
	// default-src, so without 'self' the browser refuses the frame — which looks
	// like a blocked page and reproduces in incognito, since CSP is server-sent.
	if !directiveAllowsSelf(policy, "frame-src") {
		policy = addToDirective(policy, "frame-src", "'self'")
	}

	return policy
}

// directiveAllowsSelf reports whether the named directive exists in the policy
// and already lists 'self'. The check is directive-scoped on purpose: a plain
// strings.Contains(policy, "'self'") is satisfied by default-src and would
// wrongly conclude that frame-src permits same-origin framing.
func directiveAllowsSelf(policy, directive string) bool {
	idx := strings.Index(policy, directive+" ")
	if idx == -1 {
		return false
	}
	value := policy[idx+len(directive)+1:]
	if end := strings.Index(value, ";"); end != -1 {
		value = value[:end]
	}
	for _, token := range strings.Fields(value) {
		if token == "'self'" {
			return true
		}
	}
	return false
}

// addToDirective adds a value to a specific CSP directive.
// If the directive doesn't exist, it will be added after default-src.
func addToDirective(policy, directive, value string) string {
	// Find the directive in the policy
	directivePrefix := directive + " "
	idx := strings.Index(policy, directivePrefix)

	if idx == -1 {
		// Directive not found, add it after default-src or at the beginning
		defaultSrcIdx := strings.Index(policy, "default-src ")
		if defaultSrcIdx != -1 {
			// Find the end of default-src directive (next semicolon)
			endIdx := strings.Index(policy[defaultSrcIdx:], ";")
			if endIdx != -1 {
				insertPos := defaultSrcIdx + endIdx + 1
				// Insert new directive after default-src
				return policy[:insertPos] + " " + directive + " 'self' " + value + ";" + policy[insertPos:]
			}
		}
		// Fallback: prepend the directive
		return directive + " 'self' " + value + "; " + policy
	}

	// Find the end of this directive (next semicolon or end of string)
	endIdx := strings.Index(policy[idx:], ";")

	if endIdx == -1 {
		// No semicolon found, directive goes to end of string
		return policy + " " + value
	}

	// Insert value before the semicolon
	insertPos := idx + endIdx
	return policy[:insertPos] + " " + value + policy[insertPos:]
}
