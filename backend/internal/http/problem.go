package http

import (
	"encoding/json"
	stdhttp "net/http"
)

// Problem is an RFC 9457 problem+json body. It matches core's error contract so the SDK's
// backend.request surfaces the stable `code` to the extension frontend.
type Problem struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Code   string       `json:"code"`
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError is an optional per-field validation detail.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

func writeProblem(w stdhttp.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func problem(w stdhttp.ResponseWriter, status int, code, detail string) {
	writeProblem(w, Problem{Type: "about:blank", Title: stdhttp.StatusText(status), Status: status, Code: code, Detail: detail})
}

func validationFailed(w stdhttp.ResponseWriter, fields ...FieldError) {
	writeProblem(w, Problem{
		Type:   "about:blank",
		Title:  stdhttp.StatusText(stdhttp.StatusUnprocessableEntity),
		Status: stdhttp.StatusUnprocessableEntity,
		Code:   "validation.failed",
		Detail: "one or more fields are invalid",
		Errors: fields,
	})
}
