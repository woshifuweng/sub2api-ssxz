package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var codexVersionModelPrefixes = []struct {
	prefix string
	target string
}{
	{prefix: "gpt-5.6-sol", target: "gpt-5.6-sol"},
	{prefix: "gpt-5.6-terra", target: "gpt-5.6-terra"},
	{prefix: "gpt-5.6-luna", target: "gpt-5.6-luna"},
	{prefix: "gpt-5.3-codex-spark", target: "gpt-5.3-codex-spark"},
	{prefix: "gpt-5.3-codex", target: "gpt-5.3-codex"},
	{prefix: "gpt-5.4-mini", target: "gpt-5.4-mini"},
	{prefix: "gpt-5.4-nano", target: "gpt-5.4-nano"},
	{prefix: "gpt-5.5-pro", target: "gpt-5.5-pro"},
	{prefix: "gpt-5.5", target: "gpt-5.5"},
	{prefix: "gpt-5.4", target: "gpt-5.4"},
	{prefix: "gpt-5.2", target: "gpt-5.2"},
}

type codexOAuthTransformOptions struct {
	IsCodexCLI                          bool
	IsCompact                           bool
	SkipDefaultInstructions             bool
	PreserveToolCallIDs                 bool
	OmitPromotedSystemMessagesFromInput bool
}

const (
	codexCallIDMaxLength = 64
	codexCallIDPrefix    = "fc_"
)

func normalizeCodexCallID(id string) string {
	candidate := id
	switch {
	case id == "":
		return ""
	case strings.HasPrefix(id, "fc"):
	case strings.HasPrefix(id, "call_"):
		candidate = codexCallIDPrefix + strings.TrimPrefix(id, "call_")
	default:
		candidate = codexCallIDPrefix + id
	}
	if len(candidate) <= codexCallIDMaxLength {
		return candidate
	}
	return compactCodexCallID(candidate)
}

func compactCodexCallID(id string) string {
	digest := sha256.Sum256([]byte("sub2api:codex-call-id:v1:" + id))
	encoded := hex.EncodeToString(digest[:])
	return codexCallIDPrefix + encoded[:codexCallIDMaxLength-len(codexCallIDPrefix)]
}

const codexImageGenerationFunctionToolName = "image_gen.imagegen"

const (
	codexSparkImageUnsupportedMarker = "<sub2api-codex-spark-image-unsupported>"
	codexSparkImageUnsupportedText   = codexSparkImageUnsupportedMarker + "\nThe current model is gpt-5.3-codex-spark, which does not support image generation, image editing, image input, the `image_generation` tool, or Codex `image_gen`/`$imagegen` workflows. If the user asks for image generation or image editing, clearly explain this model limitation and ask them to switch to a non-Spark Codex model such as gpt-5.3-codex or gpt-5.4. Do not claim that the local environment merely lacks image_gen tooling, and do not suggest CLI fallback as the primary fix while the model remains Spark.\n</sub2api-codex-spark-image-unsupported>"
)

var openAIChatGPTInternalUnsupportedFields = []string{
	"user",
	"metadata",
	"prompt_cache_retention",
	"safety_identifier",
	"stream_options",
}

var openAICodexOAuthUnsupportedFields = append([]string{
	"max_output_tokens",
	"max_completion_tokens",
	"temperature",
	"top_p",
	"frequency_penalty",
	"presence_penalty",
}, openAIChatGPTInternalUnsupportedFields...)

func applyCodexOAuthTransformWithOptions(reqBody map[string]any, opts codexOAuthTransformOptions) codexTransformResult {
	result := codexTransformResult{}
	// 工具续链需求会影响存储策略与 input 过滤逻辑。
	needsToolContinuation := NeedsToolContinuation(reqBody)

	model := ""
	if v, ok := reqBody["model"].(string); ok {
		model = v
	}
	normalizedModel := strings.TrimSpace(model)
	if normalizedModel != "" {
		if model != normalizedModel {
			reqBody["model"] = normalizedModel
			result.Modified = true
		}
		result.NormalizedModel = normalizedModel
	}

	if opts.IsCompact {
		if _, ok := reqBody["store"]; ok {
			delete(reqBody, "store")
			result.Modified = true
		}
		if _, ok := reqBody["stream"]; ok {
			delete(reqBody, "stream")
			result.Modified = true
		}
	} else {
		// OAuth 走 ChatGPT internal API 时，store 必须为 false；显式 true 也会强制覆盖。
		// 避免上游返回 "Store must be set to false"。
		if v, ok := reqBody["store"].(bool); !ok || v {
			reqBody["store"] = false
			result.Modified = true
		}
		if v, ok := reqBody["stream"].(bool); !ok || !v {
			reqBody["stream"] = true
			result.Modified = true
		}
	}

	// Strip parameters unsupported by ChatGPT internal Codex endpoint.
	for _, key := range openAICodexOAuthUnsupportedFields {
		if _, ok := reqBody[key]; ok {
			delete(reqBody, key)
			result.Modified = true
		}
	}

	// 请求带 reasoning 时补齐 include:["reasoning.encrypted_content"]，与真实 Codex 对齐
	// （compact 端点形态不同，单独处理，此处跳过）。
	if !opts.IsCompact && ensureCodexReasoningInclude(reqBody) {
		result.Modified = true
	}

	// 兼容遗留的 functions 和 function_call，转换为 tools 和 tool_choice
	if functionsRaw, ok := reqBody["functions"]; ok {
		if functions, k := functionsRaw.([]any); k {
			tools := make([]any, 0, len(functions))
			for _, f := range functions {
				tools = append(tools, map[string]any{
					"type":     "function",
					"function": f,
				})
			}
			reqBody["tools"] = tools
		}
		delete(reqBody, "functions")
		result.Modified = true
	}

	if fcRaw, ok := reqBody["function_call"]; ok {
		if fcStr, ok := fcRaw.(string); ok {
			// e.g. "auto", "none"
			reqBody["tool_choice"] = fcStr
		} else if fcObj, ok := fcRaw.(map[string]any); ok {
			// e.g. {"name": "my_func"}
			if name, ok := fcObj["name"].(string); ok && strings.TrimSpace(name) != "" {
				reqBody["tool_choice"] = map[string]any{
					"type": "function",
					"name": name,
				}
			}
		}
		delete(reqBody, "function_call")
		result.Modified = true
	}

	if normalizeCodexTools(reqBody) {
		result.Modified = true
	}
	if normalizeCodexToolChoice(reqBody) {
		result.Modified = true
	}

	if v, ok := reqBody["prompt_cache_key"].(string); ok {
		result.PromptCacheKey = strings.TrimSpace(v)
		if isOpenAICompatMessagesBridgeRequestBody(reqBody) {
			delete(reqBody, "prompt_cache_key")
			result.Modified = true
		}
	}

	// ChatGPT internal Codex endpoint does not accept role:"system".
	// Mirror its text into instructions because Codex OAuth requires it. Some
	// callers must also keep the guidance in input as developer (notably
	// Responses JSON object mode), while Chat Completions compatibility can
	// omit text-only messages after promoting them losslessly.
	if extractSystemMessagesFromInput(reqBody, opts.OmitPromotedSystemMessagesFromInput) {
		result.Modified = true
	}

	// instructions 处理逻辑：根据是否是 Codex CLI 分别调用不同方法
	if !opts.SkipDefaultInstructions && applyInstructions(reqBody, opts.IsCodexCLI) {
		result.Modified = true
	}
	if isCodexSparkModel(normalizedModel) && applyCodexSparkImageUnsupportedInstructions(reqBody) {
		result.Modified = true
	}
	// gpt-5.3-codex-spark rejects the image_generation tool upstream (HTTP 400,
	// param=tools); Codex CLI advertises it by default, so strip it for spark.
	if isCodexSparkModel(normalizedModel) && stripCodexSparkImageGenerationTools(reqBody) {
		result.Modified = true
	}

	// 续链场景保留 item_reference 与 id，避免 call_id 上下文丢失。
	if input, ok := reqBody["input"].([]any); ok {
		if normalizedInput, modified := normalizeCodexToolRoleMessages(input); modified {
			input = normalizedInput
			result.Modified = true
		}
		if normalizedInput, modified := normalizeCodexMessageContentText(input); modified {
			input = normalizedInput
			result.Modified = true
		}
		input = filterCodexInputWithOptions(input, codexInputFilterOptions{
			PreserveReferences: needsToolContinuation,
			PreserveCallIDs:    opts.PreserveToolCallIDs,
		})
		reqBody["input"] = input
		result.Modified = true
	} else if inputStr, ok := reqBody["input"].(string); ok {
		// ChatGPT codex endpoint requires input to be a list, not a string.
		// Convert string input to the expected message array format.
		trimmed := strings.TrimSpace(inputStr)
		if trimmed != "" {
			reqBody["input"] = []any{
				map[string]any{
					"type":    "message",
					"role":    "user",
					"content": inputStr,
				},
			}
		} else {
			reqBody["input"] = []any{}
		}
		result.Modified = true
	}

	return result
}

func codexInputAdditionalToolsContainType(rawInput any, toolType string) bool {
	input, ok := rawInput.([]any)
	if !ok || strings.TrimSpace(toolType) == "" {
		return false
	}
	for _, rawItem := range input {
		item, ok := rawItem.(map[string]any)
		if !ok || strings.TrimSpace(firstNonEmptyString(item["type"])) != "additional_tools" {
			continue
		}
		if codexToolsContainType(item["tools"], toolType) {
			return true
		}
	}
	return false
}

func codexToolsContainFunctionName(rawTools any, name string) bool {
	tools, ok := rawTools.([]any)
	if !ok || strings.TrimSpace(name) == "" {
		return false
	}
	normalizedName := strings.TrimSpace(name)
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(firstNonEmptyString(tool["type"])) != "function" {
			continue
		}
		toolName := strings.TrimSpace(firstNonEmptyString(tool["name"]))
		if toolName == "" {
			if function, ok := tool["function"].(map[string]any); ok {
				toolName = strings.TrimSpace(firstNonEmptyString(function["name"]))
			}
		}
		if toolName == normalizedName {
			return true
		}
	}
	return false
}

func normalizeKnownCodexModel(model string) (string, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", false
	}
	if isOpenAIImageGenerationModel(model) {
		return model, true
	}

	modelID := lastOpenAIModelSegment(model)

	if normalized := canonicalizeOpenAIModelAliasSpelling(modelID); normalized != "" {
		modelID = normalized
	}
	if mapped := normalizeKnownOpenAICodexModel(modelID); mapped != "" {
		return mapped, true
	}
	key := codexModelLookupKey(modelID)
	if key == "" {
		return "", false
	}
	if mapped := getNormalizedCodexModel(key); mapped != "" {
		return mapped, true
	}
	for _, item := range codexVersionModelPrefixes {
		if key == item.prefix {
			return item.target, true
		}
		suffix, ok := strings.CutPrefix(key, item.prefix+"-")
		if ok && isKnownCodexModelSuffix(suffix) {
			return item.target, true
		}
	}
	return "", false
}

func codexModelLookupKey(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		modelID = parts[len(parts)-1]
	}
	return strings.ToLower(strings.Join(strings.Fields(modelID), "-"))
}

func isKnownCodexModelSuffix(suffix string) bool {
	switch suffix {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return true
	}
	return isCodexDateSuffix(suffix)
}

func isCodexDateSuffix(suffix string) bool {
	parts := strings.Split(suffix, "-")
	if len(parts) != 3 || len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return false
	}
	for _, part := range parts {
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func hasOpenAIInputImage(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	return hasOpenAIInputImageValue(reqBody["input"]) || hasOpenAIInputImageValue(reqBody["messages"])
}

func hasOpenAIInputImageValue(value any) bool {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if hasOpenAIInputImageValue(item) {
				return true
			}
		}
	case map[string]any:
		if strings.TrimSpace(firstNonEmptyString(v["type"])) == "input_image" {
			return true
		}
		if _, ok := v["image_url"]; ok {
			return true
		}
		return hasOpenAIInputImageValue(v["content"])
	}
	return false
}

func validateCodexSparkInput(reqBody map[string]any, model string) error {
	if !isCodexSparkModel(model) || !hasOpenAIInputImage(reqBody) {
		return nil
	}
	return fmt.Errorf("model %q does not support image input", strings.TrimSpace(model))
}

func ensureOpenAIResponsesImageGenerationToolChoiceAuto(reqBody map[string]any) bool {
	if len(reqBody) == 0 || hasCodexImageGenerationFunctionTool(reqBody) || !hasOpenAIImageGenerationTool(reqBody) {
		return false
	}
	if isCodexSparkModel(firstNonEmptyString(reqBody["model"])) {
		return false
	}
	if _, ok := reqBody["tool_choice"]; ok {
		return false
	}
	reqBody["tool_choice"] = "auto"
	return true
}

func applyCodexSparkImageUnsupportedInstructions(reqBody map[string]any) bool {
	if len(reqBody) == 0 {
		return false
	}
	existing, _ := reqBody["instructions"].(string)
	if strings.Contains(existing, codexSparkImageUnsupportedMarker) {
		return false
	}
	existing = strings.TrimRight(existing, " \t\r\n")
	if strings.TrimSpace(existing) == "" {
		reqBody["instructions"] = codexSparkImageUnsupportedText
		return true
	}
	reqBody["instructions"] = existing + "\n\n" + codexSparkImageUnsupportedText
	return true
}

func validateOpenAIResponsesImageModel(reqBody map[string]any, model string) error {
	if !hasOpenAIImageGenerationTool(reqBody) {
		return nil
	}
	model = strings.TrimSpace(model)
	if !isOpenAIImageGenerationModel(model) {
		return nil
	}
	return fmt.Errorf("/v1/responses image_generation requests require a Responses-capable text model; image-only model %q is not allowed", model)
}

// extractLosslessTextFromContent returns text only when the entire content can
// be represented by an instructions string without dropping non-text parts.

func extractLosslessTextFromContent(content any) (string, bool) {
	switch v := content.(type) {
	case string:
		return v, true
	case []any:
		var b strings.Builder
		for _, part := range v {
			m, ok := part.(map[string]any)
			if !ok {
				return "", false
			}
			typeName, ok := m["type"].(string)
			if !ok || (typeName != "text" && typeName != "input_text" && typeName != "output_text") {
				return "", false
			}
			text, ok := m["text"].(string)
			if !ok {
				return "", false
			}
			_, _ = b.WriteString(text)
		}
		return b.String(), true
	default:
		return "", false
	}
}

func extractPromptLikeInstructionsFromInput(reqBody map[string]any) string {
	input, ok := reqBody["input"].([]any)
	if !ok || len(input) == 0 {
		return ""
	}
	var texts []string
	for _, item := range input {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		switch role {
		case "developer", "system":
			if text := strings.TrimSpace(extractTextFromContent(m["content"])); text != "" {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, "\n\n")
}

// defaultCodexSynthInstructions 返回合成路径在 instructions 为空时应填入的默认提示词。
//
// 按 model 选择真实 Codex CLI 的 base instructions（codex 系→GPT-5-Codex，
// gpt-5.2→GPT-5.2，gpt-5.1/gpt-5→GPT-5.1），使合成请求在提示词层面贴近真实 Codex 行为；
// 若内嵌 prompt 意外为空，回退到最小占位符以保证字段非空。

func defaultCodexSynthInstructions(model string) string {
	if instructions := strings.TrimSpace(openai.CodexBaseInstructionsForModel(model)); instructions != "" {
		return instructions
	}
	return "You are a helpful coding assistant."
}

// ensureCodexReasoningInclude 在请求带 reasoning 时补齐 include:["reasoning.encrypted_content"]。
//
// 真实 Codex 在 reasoning 存在时总会请求加密推理内容（ChatGPT/store=false 场景下用于上下文回放）。
// 该函数为加法式、幂等：仅在 include 缺失或未包含该项时追加；对非数组的异常 include 不做破坏性改写。

func ensureCodexReasoningInclude(reqBody map[string]any) bool {
	reasoning, ok := reqBody["reasoning"].(map[string]any)
	if !ok || len(reasoning) == 0 {
		return false
	}
	const encrypted = "reasoning.encrypted_content"
	switch existing := reqBody["include"].(type) {
	case nil:
		reqBody["include"] = []any{encrypted}
		return true
	case []any:
		for _, v := range existing {
			if s, ok := v.(string); ok && s == encrypted {
				return false
			}
		}
		reqBody["include"] = append(existing, encrypted)
		return true
	default:
		// include 为非预期类型时保持原样，避免破坏调用方意图。
		return false
	}
}

// applyCodexClientMetadata 在请求体补齐 client_metadata["x-codex-installation-id"]，
// 取值为账号真实的 openai_device_id（最新 Codex 在请求体携带的安装标识）。
//
// 加法式、幂等：仅在账号存在 device_id 且该键缺失时注入，绝不覆盖既有 client_metadata
// （如 turn metadata），也不伪造——无 device_id 时不写入。

func applyCodexClientMetadata(reqBody map[string]any, account *Account) bool {
	if account == nil {
		return false
	}
	deviceID := strings.TrimSpace(account.GetOpenAIDeviceID())
	if deviceID == "" {
		return false
	}
	const key = "x-codex-installation-id"
	switch existing := reqBody["client_metadata"].(type) {
	case map[string]any:
		if v, ok := existing[key].(string); ok && strings.TrimSpace(v) != "" {
			return false
		}
		existing[key] = deviceID
		reqBody["client_metadata"] = existing
		return true
	case map[string]string:
		if strings.TrimSpace(existing[key]) != "" {
			return false
		}
		next := make(map[string]any, len(existing)+1)
		for k, v := range existing {
			next[k] = v
		}
		next[key] = deviceID
		reqBody["client_metadata"] = next
		return true
	case nil:
		reqBody["client_metadata"] = map[string]any{key: deviceID}
		return true
	default:
		return false
	}
}

// applyInstructions 处理 instructions 字段：仅在 instructions 为空时填充默认值。

type codexInputFilterOptions struct {
	PreserveReferences bool
	PreserveCallIDs    bool
}

// filterCodexInput 按需过滤 item_reference 与 id。
// preserveReferences 为 true 时保持引用与 id，以满足续链请求对上下文的依赖。

func filterCodexInputWithOptions(input []any, opts codexInputFilterOptions) []any {
	filtered := make([]any, 0, len(input))
	for _, item := range input {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		typ, _ := m["type"].(string)

		// chatgpt.com codex (OAuth path) runs with store=false (forced by
		// applyCodexOAuthTransform). Replaying a reasoning item with its rs_*
		// id but no encrypted_content 404s upstream ("Item with id 'rs_...'
		// not found") — the 404 is triggered by the id lookup, not by the
		// reasoning item itself. So strip the id (always, independent of
		// PreserveReferences) yet keep the item: under store=false
		// encrypted_content is the official channel for carrying reasoning
		// context across turns, and dropping the whole item silently degrades
		// multi-turn agent reasoning. Preserve encrypted_content/content/
		// summary and every other field verbatim. Upstream additionally
		// requires a summary field — a missing one is rejected with 400
		// "Missing required parameter 'input[N].summary'" — so backfill an
		// empty array when it is absent. Contracts verified end-to-end against
		// chatgpt.com codex (gpt-5.5); see issue #1957.
		// compaction_summary items (cmp_*) are the other encrypted_content
		// carrier. Verified against the live backend: they require
		// encrypted_content (a missing one is rejected with 400), and with it
		// present the cmp_* id does not 404 whether kept or stripped. Being
		// neither reasoning nor tool calls, they flow through the generic path
		// below (id stripped when !PreserveReferences, encrypted_content
		// preserved either way), which is safe and needs no special-casing.
		if typ == "reasoning" {
			newItem := make(map[string]any, len(m))
			for key, value := range m {
				if key == "id" {
					// rs_* id replayed under store=false 404s; strip it.
					continue
				}
				newItem[key] = value
			}
			if summary, ok := newItem["summary"]; !ok || summary == nil {
				// Upstream requires a summary field; an empty array satisfies it.
				newItem["summary"] = []any{}
			}
			filtered = append(filtered, newItem)
			continue
		}

		// 仅修正真正的 tool/function call 标识，避免误改普通 message/reasoning id；
		// 若 item_reference 指向 legacy call_* 标识，则仅修正该引用本身。
		fixCallIDPrefix := func(id string) string {
			if opts.PreserveCallIDs {
				// preserve 模式尽量原样透传客户端 id 以维持 tool_use/tool_result
				// 配对，但上游对 call_id 有 64 字符硬上限，超长原样透传必然被
				// 400 拒绝（"Invalid 'input[N].call_id': string too long"）。
				// 超长时退回确定性压缩：同一逻辑 id 在 function_call 与
				// function_call_output 两侧结果一致，配对不受影响。
				if len(id) <= codexCallIDMaxLength {
					return id
				}
				return compactCodexCallID(id)
			}
			return normalizeCodexCallID(id)
		}

		if typ == "item_reference" {
			if !opts.PreserveReferences {
				continue
			}
			newItem := make(map[string]any, len(m))
			for key, value := range m {
				newItem[key] = value
			}
			if id, ok := newItem["id"].(string); ok && strings.HasPrefix(id, "call_") {
				newItem["id"] = fixCallIDPrefix(id)
			}
			filtered = append(filtered, newItem)
			continue
		}

		newItem := m
		copied := false
		// 仅在需要修改字段时创建副本，避免直接改写原始输入。
		ensureCopy := func() {
			if copied {
				return
			}
			newItem = make(map[string]any, len(m))
			for key, value := range m {
				newItem[key] = value
			}
			copied = true
		}

		if isCodexToolCallItemType(typ) {
			callID, ok := m["call_id"].(string)
			if !ok || strings.TrimSpace(callID) == "" {
				if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
					callID = id
					ensureCopy()
					newItem["call_id"] = callID
				}
			}

			if callID != "" {
				fixedCallID := fixCallIDPrefix(callID)
				if fixedCallID != callID {
					ensureCopy()
					newItem["call_id"] = fixedCallID
				}
			}
		}

		if !isCodexToolCallItemType(typ) {
			ensureCopy()
			delete(newItem, "call_id")
		}

		if codexInputItemRequiresName(typ) {
			if strings.TrimSpace(firstNonEmptyString(m["name"])) == "" {
				name := firstNonEmptyString(m["tool_name"])
				if name == "" {
					if function, ok := m["function"].(map[string]any); ok {
						name = firstNonEmptyString(function["name"])
					}
				}
				if name == "" {
					name = "tool"
				}
				ensureCopy()
				newItem["name"] = name
			}
		}

		if !opts.PreserveReferences {
			ensureCopy()
			delete(newItem, "id")
		} else if isCodexToolCallInputType(typ) {
			// 续链模式下保留 id 以维持上下文引用，但 function_call 等
			// call-input 类 item 的 id 必须以 "fc" 开头（上游校验
			// "Expected an ID that begins with 'fc'"）。item_* 形式的 id
			// 来自客户端回放，需要删除。
			// 注意：function_call_output 等 output 类的 id 无此约束，不动。
			if id, ok := m["id"].(string); ok && id != "" && !strings.HasPrefix(id, "fc") {
				ensureCopy()
				delete(newItem, "id")
			}
		} else if typ == "message" {
			// 同理，message 类 item 的 id 必须以 "msg" 开头（上游校验
			// "Expected an ID that begins with 'msg'"）。item_* 形式的 id
			// 来自客户端回放，需要删除。
			// 注意：不改写成 msg_*，改写出的 id 未必对应真实的上游对象。
			if id, ok := m["id"].(string); ok && id != "" && !strings.HasPrefix(id, "msg") {
				ensureCopy()
				delete(newItem, "id")
			}
		}

		filtered = append(filtered, newItem)
	}
	return filtered
}

func isCodexToolCallInputType(typ string) bool {
	switch typ {
	case "function_call",
		"tool_call",
		"local_shell_call",
		"tool_search_call",
		"custom_tool_call",
		"mcp_tool_call":
		return true
	default:
		return false
	}
}

func codexInputItemRequiresName(typ string) bool {
	switch strings.TrimSpace(typ) {
	case "function_call", "custom_tool_call", "mcp_tool_call":
		return true
	default:
		return false
	}
}
