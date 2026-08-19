package auth

import "time"

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	time.RFC3339,
}

func isExpired(timeStr string) bool {
	if timeStr == "" {
		return false
	}
	for _, layout := range timeLayouts {
		t, err := time.Parse(layout, timeStr)
		if err == nil {
			return time.Now().After(t)
		}
	}
	return true
}
