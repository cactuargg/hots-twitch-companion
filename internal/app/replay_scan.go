package app

// ReplayScan reports bounded, local replay correlation status.
// JSON field names are part of the dashboard API contract.
type ReplayScan struct {
 Code string `json:"code"`
 File string `json:"file,omitempty"`
 Detail string `json:"detail,omitempty"`
 CheckedAt int64 `json:"checkedAt,omitempty"`
}
