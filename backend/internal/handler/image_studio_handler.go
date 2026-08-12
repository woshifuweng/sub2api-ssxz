package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	imageStudioModel           = "gpt-image-2"
	imageStudioMaxUpload       = 32 << 20
	imageStudioCaptureMaxBytes = 64 << 20
)

type imageStudioIdempotencyPayload struct {
	TemplateID     string `json:"template_id"`
	Model          string `json:"model"`
	ProductName    string `json:"product_name"`
	SellingPoints  string `json:"selling_points"`
	Style          string `json:"style"`
	Size           string `json:"size"`
	Count          int    `json:"count"`
	Endpoint       string `json:"endpoint"`
	FileName       string `json:"file_name,omitempty"`
	FileType       string `json:"file_type,omitempty"`
	FileSize       int    `json:"file_size,omitempty"`
	FileSHA256     string `json:"file_sha256,omitempty"`
	PromptSHA256   string `json:"prompt_sha256"`
	GatewayBodySum string `json:"gateway_body_sha256"`
}

type imageStudioIdempotencyResponse struct {
	Status      int    `json:"status"`
	ContentType string `json:"content_type,omitempty"`
	Body        string `json:"body"`
}

type ImageStudioHandler struct {
	apiKeyService       *service.APIKeyService
	subscriptionService *service.SubscriptionService
	openAIGateway       *OpenAIGatewayHandler
	genService          *service.SoraGenerationService
	mediaStorage        *service.SoraMediaStorage
	cfg                 *config.Config
}

type imageStudioResponder struct {
	ctx gatewayctx.GatewayContext
}

type imageStudioRequest struct {
	TemplateID    string
	Model         string
	ProductName   string
	SellingPoints string
	Style         string
	Size          string
	Count         int
	FileName      string
	FileType      string
	FileData      []byte
}

func NewImageStudioHandler(
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	openAIGateway *OpenAIGatewayHandler,
	cfg *config.Config,
	genService *service.SoraGenerationService,
	mediaStorage *service.SoraMediaStorage,
) *ImageStudioHandler {
	return &ImageStudioHandler{
		apiKeyService:       apiKeyService,
		subscriptionService: subscriptionService,
		openAIGateway:       openAIGateway,
		genService:          genService,
		mediaStorage:        mediaStorage,
		cfg:                 cfg,
	}
}

func (g imageStudioResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g imageStudioResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

// Generate proxies the customer-facing image studio request through the
// existing OpenAI Images gateway so billing, failover, and usage records stay unified.
func (h *ImageStudioHandler) Generate(c *gin.Context) {
	h.GenerateGateway(gatewayctx.FromGin(c))
}

func (h *ImageStudioHandler) GenerateGateway(c gatewayctx.GatewayContext) {
	if h == nil || h.openAIGateway == nil || h.apiKeyService == nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusServiceUnavailable, "Image studio is not available")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	req, err := parseImageStudioRequest(c)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	apiKey, err := h.selectImageStudioAPIKey(c.Request().Context(), subject.UserID, req.Model)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	prompt := buildImageStudioPrompt(req)
	body, contentType, endpoint, err := buildImageStudioGatewayBody(req, prompt)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	operation := service.UpstreamQualityOperationImageGeneration
	if len(req.FileData) > 0 {
		operation = service.UpstreamQualityOperationImageEdit
	}
	service.SetUpstreamQualityAuditRecordContext(c, service.BuildUpstreamQualityAuditRecord(service.UpstreamQualityAuditInput{
		Route:          "/api/v1/image-studio/generate",
		Operation:      operation,
		RequestedModel: req.Model,
		ProviderName:   imageStudioProviderName(apiKey),
		Endpoint:       endpoint,
		ImageParams: service.UpstreamQualityImageParams{
			Size:  req.Size,
			Style: req.Style,
			Count: req.Count,
		},
		Prompt:         prompt,
		PromptEnhanced: true,
		Status:         "prepared",
	}))

	upstreamReq := cloneRequestForImageStudioGateway(c.Request(), endpoint, body, contentType, apiKey.Key)
	c.SetRequest(upstreamReq)
	ApplyInboundEndpointContext(c)

	if !middleware2.ApplyAPIKeyAuthWithSubscriptionContext(h.apiKeyService, h.subscriptionService, h.cfg, c) {
		return
	}

	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusServiceUnavailable, "Image request protection is not available")
		return
	}

	payload := buildImageStudioIdempotencyPayload(req, prompt, endpoint, body)
	result, err := coordinator.Execute(c.Request().Context(), service.IdempotencyExecuteOptions{
		Scope:          imageStudioIdempotencyScope(subject.UserID),
		ActorScope:     fmt.Sprintf("user:%d", subject.UserID),
		Method:         c.Request().Method,
		Route:          c.Path(),
		IdempotencyKey: c.HeaderValue("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            service.DefaultWriteIdempotencyTTL(),
	}, func(ctx context.Context) (any, error) {
		capture := newImageStudioCaptureContext(c)
		h.openAIGateway.ImagesGateway(capture)
		idempotentResponse := imageStudioIdempotencyResponse{
			Status:      capture.status,
			ContentType: strings.TrimSpace(c.Header().Get("Content-Type")),
			Body:        string(capture.bytes()),
		}
		h.persistImageStudioWork(ctx, capture, subject.UserID, apiKey.ID, req.Model, prompt)
		if !capture.success() {
			return idempotentResponse, fmt.Errorf("image studio generation failed")
		}
		return idempotentResponse, nil
	})
	if err != nil {
		if !c.ResponseWritten() {
			response.ErrorFromContext(imageStudioResponder{ctx: c}, err)
		}
		return
	}
	if result != nil && result.Replayed {
		c.SetHeader("X-Idempotency-Replayed", "true")
		if replay, ok := parseImageStudioIdempotencyResponse(result.Data); ok {
			writeImageStudioIdempotencyReplay(c, replay)
		}
	}
}

type imageStudioCaptureContext struct {
	gatewayctx.GatewayContext
	status    int
	body      bytes.Buffer
	truncated bool
}

func newImageStudioCaptureContext(ctx gatewayctx.GatewayContext) *imageStudioCaptureContext {
	return &imageStudioCaptureContext{GatewayContext: ctx}
}

func (c *imageStudioCaptureContext) SetStatus(status int) {
	if status > 0 {
		c.status = status
	}
	c.GatewayContext.SetStatus(status)
}

func (c *imageStudioCaptureContext) WriteJSON(status int, payload any) {
	if status > 0 {
		c.status = status
	}
	if body, err := json.Marshal(payload); err == nil {
		c.capture(body)
	}
	c.GatewayContext.WriteJSON(status, payload)
}

func (c *imageStudioCaptureContext) WriteBytes(status int, payload []byte) (int, error) {
	if status > 0 {
		c.status = status
	}
	c.capture(payload)
	return c.GatewayContext.WriteBytes(status, payload)
}

func (c *imageStudioCaptureContext) capture(payload []byte) {
	if c == nil || c.truncated || len(payload) == 0 {
		return
	}
	remaining := imageStudioCaptureMaxBytes - c.body.Len()
	if remaining <= 0 {
		c.truncated = true
		return
	}
	if len(payload) > remaining {
		_, _ = c.body.Write(payload[:remaining])
		c.truncated = true
		return
	}
	_, _ = c.body.Write(payload)
}

func (c *imageStudioCaptureContext) success() bool {
	return c != nil && c.status >= http.StatusOK && c.status < http.StatusMultipleChoices && !c.truncated
}

func (c *imageStudioCaptureContext) bytes() []byte {
	if c == nil {
		return nil
	}
	return c.body.Bytes()
}

func (h *ImageStudioHandler) persistImageStudioWork(ctx context.Context, capture *imageStudioCaptureContext, userID int64, apiKeyID int64, model string, prompt string) {
	if h == nil || h.genService == nil || !capture.success() {
		return
	}
	model = normalizeImageStudioModel(model)
	urls := extractImageStudioResultURLs(capture.bytes())
	storageType := service.SoraStorageTypeUpstream
	var fileSizeBytes int64
	if len(urls) > 0 && h.mediaStorage != nil && h.mediaStorage.Enabled() {
		storedPaths, err := h.mediaStorage.StoreFromURLs(ctx, "image", urls)
		if err == nil && imageStudioAllLocalImagePaths(storedPaths) {
			urls = storedPaths
			storageType = service.SoraStorageTypeLocal
			if totalSize, sizeErr := h.mediaStorage.TotalSizeByRelativePaths(storedPaths); sizeErr == nil {
				fileSizeBytes = totalSize
			}
		} else if err == nil && len(storedPaths) > 0 {
			_ = h.mediaStorage.DeleteByRelativePaths(storedPaths)
		}
	}
	if len(urls) == 0 && h.mediaStorage != nil && h.mediaStorage.Enabled() {
		localPaths, totalSize, err := h.mediaStorage.StoreBase64Images(ctx, extractImageStudioResultBase64(capture.bytes()))
		if err == nil && len(localPaths) > 0 {
			urls = localPaths
			storageType = service.SoraStorageTypeLocal
			fileSizeBytes = totalSize
		}
	}
	if len(urls) == 0 {
		return
	}
	var apiKeyIDPtr *int64
	if apiKeyID > 0 {
		apiKeyIDPtr = &apiKeyID
	}
	_, _ = h.genService.CreateCompletedImageWork(ctx, userID, apiKeyIDPtr, model, prompt, urls, storageType, nil, fileSizeBytes)
}

func imageStudioAllLocalImagePaths(urls []string) bool {
	if len(urls) == 0 {
		return false
	}
	for _, raw := range urls {
		cleaned := path.Clean(strings.TrimSpace(raw))
		if !strings.HasPrefix(cleaned, "/image/") {
			return false
		}
	}
	return true
}

func parseImageStudioRequest(c gatewayctx.GatewayContext) (*imageStudioRequest, error) {
	if c == nil || c.Request() == nil {
		return nil, fmt.Errorf("missing request")
	}
	req := c.Request()
	if err := req.ParseMultipartForm(imageStudioMaxUpload); err != nil {
		return nil, fmt.Errorf("invalid form data")
	}

	formValue := func(name string) string {
		if req.MultipartForm == nil || req.MultipartForm.Value == nil {
			return ""
		}
		values := req.MultipartForm.Value[name]
		if len(values) == 0 {
			return ""
		}
		return strings.TrimSpace(values[0])
	}

	out := &imageStudioRequest{
		TemplateID:    normalizeImageStudioTemplate(formValue("template_id")),
		Model:         normalizeImageStudioModel(formValue("model")),
		ProductName:   truncateStudioText(formValue("product_name"), 120),
		SellingPoints: truncateStudioText(formValue("selling_points"), 600),
		Style:         truncateStudioText(formValue("style"), 180),
		Size:          normalizeImageStudioSize(formValue("size")),
		Count:         normalizeImageStudioCount(formValue("count")),
	}

	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		files := req.MultipartForm.File["image"]
		if len(files) > 0 && files[0] != nil {
			fileHeader := files[0]
			if fileHeader.Size > imageStudioMaxUpload {
				return nil, fmt.Errorf("image is too large")
			}
			file, err := fileHeader.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read image")
			}
			defer func() { _ = file.Close() }()
			data, err := io.ReadAll(io.LimitReader(file, imageStudioMaxUpload+1))
			if err != nil {
				return nil, fmt.Errorf("failed to read image")
			}
			if len(data) > imageStudioMaxUpload {
				return nil, fmt.Errorf("image is too large")
			}
			detectedType, ok := detectAllowedImageStudioFileType(data)
			if !ok {
				return nil, fmt.Errorf("unsupported image file type")
			}
			out.FileName = sanitizeStudioFileName(fileHeader.Filename)
			out.FileType = detectedType
			out.FileData = data
		}
	}

	return out, nil
}

func (h *ImageStudioHandler) selectImageStudioAPIKey(ctx context.Context, userID int64, model string) (*service.APIKey, error) {
	model = normalizeImageStudioModel(model)
	keys, _, err := h.apiKeyService.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{Status: service.StatusAPIKeyActive})
	if err != nil {
		return nil, fmt.Errorf("failed to load API keys")
	}
	for i := range keys {
		key := &keys[i]
		if key == nil || !key.IsActive() || strings.TrimSpace(key.Key) == "" {
			continue
		}
		if !key.AllowsModel(model) {
			continue
		}
		if key.Group != nil && key.Group.Platform == service.PlatformOpenAI {
			return key, nil
		}
	}
	return nil, fmt.Errorf("please create an active OpenAI API key for the selected image model before using image studio")
}

func buildImageStudioGatewayBody(req *imageStudioRequest, prompt string) ([]byte, string, string, error) {
	if req == nil {
		return nil, "", "", fmt.Errorf("missing request")
	}
	if len(req.FileData) > 0 {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		fields := map[string]string{
			"model":  normalizeImageStudioModel(req.Model),
			"prompt": prompt,
			"size":   req.Size,
			"n":      fmt.Sprintf("%d", req.Count),
		}
		for name, value := range fields {
			if err := writer.WriteField(name, value); err != nil {
				return nil, "", "", err
			}
		}
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, escapeMultipartFileName(req.FileName)))
		if req.FileType != "" {
			partHeader.Set("Content-Type", req.FileType)
		}
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, "", "", err
		}
		if _, err := part.Write(req.FileData); err != nil {
			return nil, "", "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", "", err
		}
		return body.Bytes(), writer.FormDataContentType(), "/v1/images/edits", nil
	}

	payload := map[string]any{
		"model":  normalizeImageStudioModel(req.Model),
		"prompt": prompt,
		"size":   req.Size,
		"n":      req.Count,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", "", err
	}
	return body, "application/json", "/v1/images/generations", nil
}

func normalizeImageStudioModel(value string) string {
	value = truncateStudioText(value, 120)
	if value == "" {
		return imageStudioModel
	}
	return value
}

func cloneRequestForImageStudioGateway(req *http.Request, endpoint string, body []byte, contentType string, apiKey string) *http.Request {
	next := req.Clone(req.Context())
	next.Method = http.MethodPost
	next.Body = io.NopCloser(bytes.NewReader(body))
	next.ContentLength = int64(len(body))
	next.Header = req.Header.Clone()
	next.Header.Set("Authorization", "Bearer "+apiKey)
	next.Header.Set("Content-Type", contentType)
	next.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	if next.URL != nil {
		copiedURL := *next.URL
		copiedURL.Path = endpoint
		copiedURL.RawPath = ""
		copiedURL.RawQuery = ""
		next.URL = &copiedURL
	}
	next.RequestURI = endpoint
	return next
}

func buildImageStudioPrompt(req *imageStudioRequest) string {
	templateInstruction := map[string]string{
		"background": "Create an ecommerce main product image. Keep the uploaded product exactly consistent in shape, color, material, logo, and packaging details. Replace the background with a clean commercial background.",
		"white":      "Create a clean white background ecommerce product image. Preserve the product exactly. Improve edges, lighting, clarity, and realistic commercial texture.",
		"scene":      "Create a realistic ecommerce lifestyle scene using the uploaded product as the hero item. Keep the product design unchanged and make it suitable for a product detail page.",
		"poster":     "Create a fresh social media product poster composition. Keep the product accurate, polished, and commercially usable. Do not add unreadable text or watermarks.",
	}
	instruction := templateInstruction[req.TemplateID]
	if instruction == "" {
		instruction = templateInstruction["background"]
	}
	product := req.ProductName
	if product == "" {
		product = "the product"
	}
	points := req.SellingPoints
	if points == "" {
		points = "clear product value, premium texture, suitable for ecommerce conversion"
	}
	style := req.Style
	if style == "" {
		style = "clean studio commercial photography"
	}
	return strings.Join([]string{
		instruction,
		"Product name: " + product + ".",
		"Selling points: " + points + ".",
		"Visual style: " + style + ".",
		"Use professional commercial photography lighting, sharp product edges, realistic shadows, clean composition, no watermark, no extra logos, no distorted text, no fake accessories, no change to the product structure.",
	}, "\n")
}

func normalizeImageStudioTemplate(value string) string {
	switch strings.TrimSpace(value) {
	case "background", "white", "scene", "poster":
		return value
	default:
		return "background"
	}
}

func normalizeImageStudioSize(value string) string {
	switch strings.TrimSpace(value) {
	case "1024x1024", "1024x1536", "1536x1024":
		return value
	default:
		return "1024x1024"
	}
}

func normalizeImageStudioCount(value string) int {
	switch strings.TrimSpace(value) {
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	default:
		return 1
	}
}

func truncateStudioText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

func sanitizeStudioFileName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	if value == "" {
		return "image.png"
	}
	return value
}

func escapeMultipartFileName(value string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(sanitizeStudioFileName(value))
}

func imageStudioProviderName(apiKey *service.APIKey) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func imageStudioIdempotencyScope(userID int64) string {
	return fmt.Sprintf("user:%d:image_studio_generate", userID)
}

func buildImageStudioIdempotencyPayload(req *imageStudioRequest, prompt string, endpoint string, gatewayBody []byte) imageStudioIdempotencyPayload {
	payload := imageStudioIdempotencyPayload{
		Endpoint:       endpoint,
		PromptSHA256:   sha256Hex([]byte(prompt)),
		GatewayBodySum: sha256Hex(gatewayBody),
	}
	if req == nil {
		return payload
	}
	payload.TemplateID = req.TemplateID
	payload.Model = req.Model
	payload.ProductName = req.ProductName
	payload.SellingPoints = req.SellingPoints
	payload.Style = req.Style
	payload.Size = req.Size
	payload.Count = req.Count
	payload.FileName = req.FileName
	payload.FileType = req.FileType
	payload.FileSize = len(req.FileData)
	payload.FileSHA256 = sha256Hex(req.FileData)
	return payload
}

func parseImageStudioIdempotencyResponse(data any) (imageStudioIdempotencyResponse, bool) {
	switch value := data.(type) {
	case imageStudioIdempotencyResponse:
		return value, true
	case *imageStudioIdempotencyResponse:
		if value == nil {
			return imageStudioIdempotencyResponse{}, false
		}
		return *value, true
	case map[string]any:
		out := imageStudioIdempotencyResponse{}
		if status, ok := value["status"].(float64); ok {
			out.Status = int(status)
		}
		if contentType, ok := value["content_type"].(string); ok {
			out.ContentType = contentType
		}
		if body, ok := value["body"].(string); ok {
			out.Body = body
		}
		return out, out.Status > 0
	default:
		return imageStudioIdempotencyResponse{}, false
	}
}

func writeImageStudioIdempotencyReplay(c gatewayctx.GatewayContext, replay imageStudioIdempotencyResponse) {
	if c == nil {
		return
	}
	status := replay.Status
	if status <= 0 {
		status = http.StatusOK
	}
	if strings.TrimSpace(replay.ContentType) != "" {
		c.SetHeader("Content-Type", replay.ContentType)
	} else {
		c.SetHeader("Content-Type", "application/json")
	}
	_, _ = c.WriteBytes(status, []byte(replay.Body))
}

func detectAllowedImageStudioFileType(data []byte) (string, bool) {
	if len(data) == 0 {
		return "", false
	}
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	switch detected {
	case "image/png", "image/jpeg", "image/webp":
		return detected, true
	default:
		return "", false
	}
}

func sha256Hex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func extractImageStudioResultURLs(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	var payload struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	urls := make([]string, 0, len(payload.Data))
	seen := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		u := strings.TrimSpace(item.URL)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}
	return urls
}

func extractImageStudioResultBase64(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	var payload struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	images := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		raw := strings.TrimSpace(item.B64JSON)
		if raw == "" {
			continue
		}
		images = append(images, raw)
	}
	return images
}
