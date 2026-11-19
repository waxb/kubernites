package live

import "time"

func SetUsageLimit(d time.Duration) func() {
	old := usageLimit
	usageLimit = d
	return func() { usageLimit = old }
}
