package service

// resolveOpenAIForwardModel determines the upstream model for OpenAI-compatible
// forwarding. Group-level default mapping only applies when the account itself
// did not match any explicit model_mapping rule.
func resolveOpenAIForwardModel(account *Account, requestedModel, defaultMappedModel string) string {
	mappedModel, _ := resolveOpenAIForwardModelWithMatch(account, requestedModel, defaultMappedModel)
	return mappedModel
}

// resolveOpenAIForwardModelWithMatch also reports whether account model_mapping
// selected the upstream model. Explicit mappings must survive generic Codex
// model normalization, including identity mappings.
func resolveOpenAIForwardModelWithMatch(account *Account, requestedModel, defaultMappedModel string) (string, bool) {
	if account == nil {
		if defaultMappedModel != "" {
			return defaultMappedModel, false
		}
		return requestedModel, false
	}

	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if !matched && defaultMappedModel != "" {
		return defaultMappedModel, false
	}
	return mappedModel, matched
}
