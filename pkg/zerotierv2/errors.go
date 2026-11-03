package zerotierv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a non-2xx response from Central V2.
type APIError struct {
	StatusCode int
	Message    string
	Details    []string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	if len(e.Details) > 0 {
		msg += " (" + strings.Join(e.Details, "; ") + ")"
	}
	return fmt.Sprintf("Central V2 API returned %d: %s", e.StatusCode, msg)
}

// newAPIError builds an APIError from a response the generated client did not
// decode as a success. Central V2 error bodies are {"message": ...}, with
// {"code", "message", "details": [{"field", "message"}]} for 400s.
func newAPIError(resp *http.Response, body []byte) error {
	if resp == nil {
		return errors.New("Central V2 API returned no response")
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}

	var parsed struct {
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		apiErr.Message = parsed.Message
		for _, d := range parsed.Details {
			apiErr.Details = append(apiErr.Details, strings.TrimSpace(d.Field+": "+d.Message))
		}
	} else {
		apiErr.Message = strings.TrimSpace(string(body))
	}

	return apiErr
}

// isGone reports whether err means the object no longer exists.
//
// Central V2 checks permissions before it looks an object up, and deleting an
// object removes its permission tuples, so reading a deleted network group,
// network, service account or API key returns 403 rather than 404. Members
// are looked up by the handler and return 404.
func isGone(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusForbidden
}

// check returns nil when the request succeeded (json200 != nil), otherwise
// the transport error or the API error.
func check[T any](err error, json200 *T, resp *http.Response, body []byte) (*T, error) {
	if err != nil {
		return nil, err
	}
	if json200 == nil {
		if resp != nil && resp.StatusCode == http.StatusOK {
			return nil, fmt.Errorf("Central V2 API returned an unexpected response body: %s", strings.TrimSpace(string(body)))
		}
		return nil, newAPIError(resp, body)
	}
	return json200, nil
}

// checkNoBody is check for operations whose success body is not needed.
func checkNoBody(err error, resp *http.Response, body []byte) error {
	if err != nil {
		return err
	}
	if resp == nil || resp.StatusCode/100 != 2 {
		return newAPIError(resp, body)
	}
	return nil
}
