package dashboard

import (
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "errors"
 "io"
 "net/http"
)

const MaxJSONRequestBytes int64 = 16384

// ReadJSONRequest rejects unknown fields, oversized bodies, and trailing values.
// The caller supplies its localized too-large error.
func ReadJSONRequest(r *http.Request, v any, tooLargeMessage string) error {
 if r.ContentLength > MaxJSONRequestBytes { return errors.New(tooLargeMessage) }
 r.Body = http.MaxBytesReader(nil, r.Body, MaxJSONRequestBytes)
 dec := json.NewDecoder(r.Body)
 dec.DisallowUnknownFields()
 if err := dec.Decode(v); err != nil { return err }
 var extra any
 if err := dec.Decode(&extra); err != io.EOF {
  if err == nil { return errors.New("multiple JSON values in request body") }
  return err
 }
 return nil
}

func JSONOutput(w http.ResponseWriter, code int, v any) {
 w.Header().Set("Content-Type", "application/json; charset=utf-8")
 w.Header().Set("Cache-Control", "no-store")
 w.WriteHeader(code)
 _ = json.NewEncoder(w).Encode(v)
}

func RequestError(w http.ResponseWriter, err error) {
 JSONOutput(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

// RandomSegment returns a 192-bit cryptographically random URL capability.
func RandomSegment() (string, error) {
 b := make([]byte, 24)
 if _, err := rand.Read(b); err != nil { return "", err }
 return hex.EncodeToString(b), nil
}
