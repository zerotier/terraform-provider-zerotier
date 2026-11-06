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
	Code       int // Central's error code, when the body has one
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
	if e.StatusCode == http.StatusPaymentRequired {
		// Central sends no detail with a 402. It means a plan feature is
		// missing (code 7001) or a plan limit would be exceeded (code 7002).
		msg += ". The organization's plan doesn't allow this change: either it doesn't include the feature " +
			"(for example public networks, custom routes or custom DNS), or the change would exceed a plan limit, " +
			"such as the number of authorized devices"
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
		Code    int    `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		apiErr.Code = parsed.Code
		apiErr.Message = parsed.Message
		for _, d := range parsed.Details {
			apiErr.Details = append(apiErr.Details, strings.TrimSpace(d.Field+": "+d.Message))
		}
	} else {
		apiErr.Message = strings.TrimSpace(string(body))
	}

	return apiErr
}

// Central V2's "unknown <object>" error codes, returned with a 400
// (core/common/errorcodes.go in central-v2).
var unknownObjectCodes = map[int]bool{
	2001: true, // org
	3001: true, // network group
	4001: true, // network
	4101: true, // network member
	5101: true, // service account
	5201: true, // api key
}

// isGone reports whether err means the object no longer exists. Central V2
// says so in three ways:
//
//   - 403: it checks permissions before looking an object up, and deleting an
//     object removes its permission tuples, so a deleted network group,
//     network, service account or API key (or a member of a deleted network)
//     reads as forbidden.
//   - 400 with an "unknown <object>" code: a member removed from a network
//     that still exists (code 4101).
//   - 404.
func isGone(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
	case http.StatusNotFound, http.StatusForbidden:
		return true
	case http.StatusBadRequest:
		return unknownObjectCodes[apiErr.Code]
	}
	return false
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
