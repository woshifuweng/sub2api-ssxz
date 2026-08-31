package service

import (
	"os"
	"strconv"
	"strings"
)

const backgroundJobsEnabledEnv = "BACKGROUND_JOBS_ENABLED"

// backgroundJobsEnabled keeps production behavior unchanged when the variable
// is absent. An explicitly invalid value fails closed so staging cannot
// accidentally start autonomous workers because of a typo.
func backgroundJobsEnabled() bool {
	raw, ok := os.LookupEnv(backgroundJobsEnabledEnv)
	if !ok {
		return true
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && enabled
}

func startBackgroundJob(start func()) {
	if start != nil && backgroundJobsEnabled() {
		start()
	}
}
