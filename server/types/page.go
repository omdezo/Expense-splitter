package types

// Page is the envelope for paginated lists: Total is the full match count
// (ignoring limit/offset), Items is the current window.
type Page struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Items  any   `json:"items"`
}

const (
	PageDefaultLimit = 50
	PageMaxLimit     = 200
)
