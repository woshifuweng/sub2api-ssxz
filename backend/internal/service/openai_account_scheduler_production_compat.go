package service

func buildOpenAITieredSelectionOrder(
	candidates []openAIAccountCandidateScore,
	topK int,
	req OpenAIAccountScheduleRequest,
) []openAIAccountCandidateScore {
	if len(candidates) <= 1 {
		return append([]openAIAccountCandidateScore(nil), candidates...)
	}
	if topK <= 0 {
		topK = 1
	}

	tierBuckets := [3][]openAIAccountCandidateScore{}
	for _, candidate := range candidates {
		tier := candidate.readyTier
		if tier < 0 {
			tier = 0
		}
		if tier > 2 {
			tier = 2
		}
		tierBuckets[tier] = append(tierBuckets[tier], candidate)
	}

	order := make([]openAIAccountCandidateScore, 0, len(candidates))
	for tier := 0; tier < len(tierBuckets); tier++ {
		if len(tierBuckets[tier]) == 0 {
			continue
		}
		ranked := selectTopKOpenAICandidates(tierBuckets[tier], topK)
		order = append(order, buildOpenAIWeightedSelectionOrder(ranked, req)...)
	}
	return order
}

func adaptiveOpenAISelectionTopK(baseTopK int, candidateCount int, loadSkew float64) int {
	return adaptiveSchedulerSelectionTopK(baseTopK, candidateCount, loadSkew)
}
