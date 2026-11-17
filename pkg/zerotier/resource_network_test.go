package zerotier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerotier/go-ztcentral"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

// fakeCentral records request bodies by "METHOD path" and replies with the
// configured status and body.
type fakeCentral struct {
	mu        sync.Mutex
	requests  map[string][]map[string]interface{}
	responses map[string]fakeResponse
}

type fakeResponse struct {
	status int
	body   string
}

func newFakeCentral(t *testing.T, responses map[string]fakeResponse) (*fakeCentral, *ztcentral.Client) {
	t.Helper()

	f := &fakeCentral{requests: map[string][]map[string]interface{}{}, responses: responses}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)

	orig := ztcentral.BaseURLV1
	ztcentral.BaseURLV1 = srv.URL
	defer func() { ztcentral.BaseURLV1 = orig }()

	c, err := ztcentral.NewClient("test-token")
	require.NoError(t, err)

	return f, c
}

func (f *fakeCentral) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path

	body := map[string]interface{}{}
	raw, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(raw, &body)

	f.mu.Lock()
	f.requests[key] = append(f.requests[key], body)
	resp, ok := f.responses[key]
	f.mu.Unlock()

	if !ok {
		resp = fakeResponse{http.StatusNotFound, `{"type":"not_found","message":"no fake for ` + key + `"}`}
	}

	w.WriteHeader(resp.status)
	_, _ = io.WriteString(w, resp.body)
}

func (f *fakeCentral) bodies(key string) []map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[key]
}

const fakeNetworkJSON = `{"id":"8056c2e21c000001","rulesSource":"drop;","description":"d","config":{"name":"n","mtu":1400,"multicastLimit":64}}`

func TestResourceNetworkCreateAppliesFullConfig(t *testing.T) {
	f, c := newFakeCentral(t, map[string]fakeResponse{
		"POST /network":                  {http.StatusOK, `{"id":"8056c2e21c000001","rulesSource":"accept;","config":{"name":"n","mtu":2800,"multicastLimit":32}}`},
		"POST /network/8056c2e21c000001": {http.StatusOK, fakeNetworkJSON},
	})

	d := schema.TestResourceDataRaw(t, NetworkSchema, map[string]interface{}{
		"name":            "n",
		"mtu":             1400,
		"multicast_limit": 64,
		"flow_rules":      "drop;",
	})

	diags := resourceNetworkCreate(context.Background(), d, c)
	require.False(t, diags.HasError(), "%v", diags)

	updates := f.bodies("POST /network/8056c2e21c000001")
	require.NotEmpty(t, updates, "create must follow up with an update")

	cfg := updates[0]["config"].(map[string]interface{})
	assert.EqualValues(t, 1400, cfg["mtu"])
	assert.EqualValues(t, 64, cfg["multicastLimit"])

	assert.Equal(t, "8056c2e21c000001", d.Id())
	assert.Equal(t, 1400, d.Get("mtu"))
	assert.Equal(t, 64, d.Get("multicast_limit"))
	assert.Equal(t, "drop;", d.Get("flow_rules"))
}

func TestResourceNetworkCreateKeepsIDWhenUpdateFails(t *testing.T) {
	_, c := newFakeCentral(t, map[string]fakeResponse{
		"POST /network":                  {http.StatusOK, `{"id":"8056c2e21c000001","config":{"name":"n"}}`},
		"POST /network/8056c2e21c000001": {http.StatusLocked, `{"type":"migration_locked","message":"network controller is in migration-lock (read-only) mode"}`},
	})

	d := schema.TestResourceDataRaw(t, NetworkSchema, map[string]interface{}{"name": "n"})

	diags := resourceNetworkCreate(context.Background(), d, c)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary+diags[0].Detail, "migration-lock")
	assert.Equal(t, "8056c2e21c000001", d.Id(), "network must stay in state so it is not orphaned")
}

func TestResourceNetworkReadNotFoundClearsState(t *testing.T) {
	_, c := newFakeCentral(t, nil)

	d := schema.TestResourceDataRaw(t, NetworkSchema, map[string]interface{}{})
	d.SetId("8056c2e21c000001")

	diags := resourceNetworkRead(context.Background(), d, c)
	assert.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "", d.Id())
}

func TestResourceNetworkDeleteNotFoundSucceeds(t *testing.T) {
	_, c := newFakeCentral(t, nil)

	d := schema.TestResourceDataRaw(t, NetworkSchema, map[string]interface{}{})
	d.SetId("8056c2e21c000001")

	diags := resourceNetworkDelete(context.Background(), d, c)
	assert.False(t, diags.HasError(), "%v", diags)
}

func TestNetworkToTerraformToleratesMissingFields(t *testing.T) {
	d := schema.TestResourceDataRaw(t, NetworkSchema, map[string]interface{}{})

	assert.NotPanics(t, func() {
		networkToTerraform(d, &spec.Network{Id: stringPtr("8056c2e21c000001")})
	})
	assert.Equal(t, "8056c2e21c000001", d.Id())
}

func TestNetworkMTUValidation(t *testing.T) {
	validate := NetworkSchema["mtu"].ValidateDiagFunc

	assert.False(t, validate(2800, nil).HasError())
	assert.True(t, validate(1279, nil).HasError())
	assert.True(t, validate(10001, nil).HasError())
}
