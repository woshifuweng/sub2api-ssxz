package service

// ReplaceModelInBody keeps OpenAI-compatible handlers on the gateway service
// boundary while reusing the shared order-preserving JSON model replacement.
func (s *OpenAIGatewayService) ReplaceModelInBody(body []byte, newModel string) []byte {
	return ReplaceModelInBody(body, newModel)
}
