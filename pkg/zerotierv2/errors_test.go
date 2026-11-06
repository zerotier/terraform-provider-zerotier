package zerotierv2

import (
	"net/http"
	"strings"
	"testing"
)

func TestNewAPIError(t *testing.T) {
	resp := func(status int) *http.Response { return &http.Response{StatusCode: status} }

	tests := []struct {
		name     string
		status   int
		body     string
		want     []string
		wantGone bool
	}{
		{
			name:   "402 covers missing features and exceeded limits",
			status: http.StatusPaymentRequired,
			body:   `{"message":"402 Payment Required"}`,
			want:   []string{"402", "doesn't include the feature", "public networks", "exceed a plan limit", "authorized devices"},
		},
		{
			name:     "removed member",
			status:   http.StatusBadRequest,
			body:     `{"code":4101,"message":"unknown network member"}`,
			want:     []string{"400", "unknown network member"},
			wantGone: true,
		},
		{
			name:   "other 400 keeps field details",
			status: http.StatusBadRequest,
			body:   `{"code":1000,"message":"invalid request","details":[{"field":"name","message":"too short"}]}`,
			want:   []string{"invalid request", "name: too short"},
		},
		{
			name:     "deleted object",
			status:   http.StatusForbidden,
			body:     `{"message":"403 Forbidden"}`,
			want:     []string{"403"},
			wantGone: true,
		},
		{
			name:   "non-JSON body",
			status: http.StatusBadGateway,
			body:   "upstream unavailable",
			want:   []string{"502", "upstream unavailable"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newAPIError(resp(tc.status), []byte(tc.body))
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
			if got := isGone(err); got != tc.wantGone {
				t.Errorf("isGone = %v, want %v", got, tc.wantGone)
			}
		})
	}
}
