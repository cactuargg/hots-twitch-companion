package transport

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestHTTPErrorRateLimitResetPriority(t *testing.T) {
	now:=time.Now()
	response:=&http.Response{StatusCode:429,Header:make(http.Header)}
	response.Header.Set("Retry-After","2")
	reset:=now.Add(40*time.Second).Unix()
	response.Header.Set("Ratelimit-Reset",strconv.FormatInt(reset,10))
	err:=ResponseHTTPError(response,"too many")
	var got *HTTPError
	if !errors.As(err,&got)||got.Status!=429||got.Message!="too many"||got.RetryAt!=reset*1000 {
		t.Fatalf("Twitch rate limit was lost: %+v",got)
	}
	if got.Error()!="Twitch HTTP 429: too many"{t.Fatalf("changed HTTP error text: %s",got.Error())}
}

func TestHTTPErrorRetryAfterDatesAndNonRateLimits(t *testing.T) {
	at:=time.Now().Add(2*time.Hour).UTC().Truncate(time.Second)
	response:=&http.Response{StatusCode:429,Header:make(http.Header)}
	response.Header.Set("Retry-After",at.Format(http.TimeFormat))
	err:=ResponseHTTPError(response,"slow").(*HTTPError)
	if err.RetryAt!=at.UnixMilli(){t.Fatalf("http-date retry lost: %d vs %d",err.RetryAt,at.UnixMilli())}
	response.StatusCode=400
	got:=ResponseHTTPError(response,"bad request").(*HTTPError)
	if got.RetryAt!=0{t.Fatal("non-429 response must not schedule retry")}
}
