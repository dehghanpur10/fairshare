package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/service"
)

const maxBodyBytes = 1 << 20

// problem is an RFC 9457 problem details body.
type problem struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Status    int               `json:"status"`
	Detail    string            `json:"detail,omitempty"`
	Errors    map[string]string `json:"errors,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// requestError is a client mistake caught before the service layer runs,
// such as a malformed body or a missing token.
type requestError struct {
	status int
	detail string
}

func (e *requestError) Error() string { return e.detail }

func badRequest(format string, args ...any) error {
	return &requestError{status: http.StatusBadRequest, detail: fmt.Sprintf(format, args...)}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError maps an error to a problem response. Anything unrecognised is a
// 500 whose detail stays in the logs, never in the response.
func (s *server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	p := problem{Type: "about:blank", RequestID: requestIDFrom(r.Context())}

	var validation *service.ValidationError
	var reqErr *requestError
	switch {
	case errors.As(err, &validation):
		p.Status, p.Title, p.Errors = http.StatusUnprocessableEntity, "Validation failed", validation.Fields
	case errors.As(err, &reqErr):
		p.Status, p.Detail = reqErr.status, reqErr.detail
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidToken):
		p.Status, p.Detail = http.StatusUnauthorized, "Missing, invalid or expired credentials."
		if errors.Is(err, service.ErrInvalidCredentials) {
			p.Detail = "Invalid email or password."
		}
		w.Header().Set("WWW-Authenticate", `Bearer realm="fairshare"`)
	case errors.Is(err, service.ErrForbidden):
		p.Status, p.Detail = http.StatusForbidden, "You do not have permission to do that."
	case errors.Is(err, service.ErrNotFound):
		p.Status = http.StatusNotFound
	case errors.Is(err, service.ErrEmailTaken), errors.Is(err, service.ErrAlreadyMember):
		p.Status, p.Detail = http.StatusConflict, err.Error()
	default:
		p.Status = http.StatusInternalServerError
		s.log.ErrorContext(r.Context(), "request failed",
			"error", err, "method", r.Method, "path", r.URL.Path, "request_id", p.RequestID)
	}
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// decodeJSON reads a single JSON object into dst, rejecting unknown fields,
// trailing data and bodies over maxBodyBytes.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return &requestError{status: http.StatusUnsupportedMediaType, detail: "Content-Type must be application/json."}
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var tooLarge *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return badRequest("Request body must not be empty.")
		case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
			return badRequest("Request body is not valid JSON.")
		case errors.As(err, &typeErr) && typeErr.Field != "":
			return badRequest("Field %q has the wrong type.", typeErr.Field)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return badRequest("Unknown field %s.", strings.TrimPrefix(err.Error(), "json: unknown field "))
		case errors.As(err, &tooLarge):
			return &requestError{status: http.StatusRequestEntityTooLarge, detail: "Request body is too large."}
		default:
			return badRequest("Request body could not be decoded.")
		}
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return badRequest("Request body must contain a single JSON object.")
	}
	return nil
}
