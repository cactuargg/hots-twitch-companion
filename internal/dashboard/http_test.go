package dashboard

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJSONRequest(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		fail       bool
	}{
		{"valid", `{"name":"okay"}`, false},
		{"unknown", `{"other":1}`, true},
		{"multiple", `{"name":"okay"} {"name":"again"}`, true},
		{"trailing", `{"name":"okay"} garbage`, true},
		{"oversized", strings.Repeat("x", 16385), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			var value struct {
				Name string `json:"name"`
			}
			err := ReadJSONRequest(req, &value, "too large")
			if (err != nil) != tt.fail {
				t.Fatalf("err=%v fail=%v", err, tt.fail)
			}
		})
	}
}
func TestJSONOutput(t *testing.T) {
	w := httptest.NewRecorder()
	JSONOutput(w, 201, map[string]bool{"ok": true})
	if w.Code != 201 || w.Header().Get("Cache-Control") != "no-store" || !bytes.Contains(w.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("response: %+v", w)
	}
}
func TestRandomSegment(t *testing.T) {
	a, err := RandomSegment()
	if err != nil {
		t.Fatal(err)
	}
	b, err := RandomSegment()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 48 || a == b {
		t.Fatalf("invalid URL secret length or duplicate")
	}
}
