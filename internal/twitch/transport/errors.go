package transport

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// HTTPError retains Twitch rate-limit data for durable settlement retries.
type HTTPError struct {
	Status  int
	Message string
	RetryAt int64
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Twitch HTTP %d: %s", e.Status, e.Message)
}
func ResponseHTTPError(response *http.Response, message string) error {
	failure := &HTTPError{Status: response.StatusCode, Message: message}
	if response.StatusCode == 429 {
		if seconds, err := strconv.ParseInt(response.Header.Get("Retry-After"), 10, 64); err == nil && seconds >= 0 && seconds < 86400 {
			failure.RetryAt = time.Now().Add(time.Duration(seconds) * time.Second).UnixMilli()
		} else if at, err := http.ParseTime(response.Header.Get("Retry-After")); err == nil {
			failure.RetryAt = at.UnixMilli()
		}
		if reset, err := strconv.ParseInt(response.Header.Get("Ratelimit-Reset"), 10, 64); err == nil && reset > 0 && reset < 1<<40 {
			failure.RetryAt = max(failure.RetryAt, reset*1000)
		}
	}
	return failure
}

