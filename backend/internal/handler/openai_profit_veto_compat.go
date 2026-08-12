package handler

func recordOpenAIProfitVeto(failed map[int64]struct{}, accountID int64, count *int) bool {
	if _, exists := failed[accountID]; exists {
		return true
	}
	failed[accountID] = struct{}{}
	*count = *count + 1
	return *count < maxProfitVetoAttempts
}
