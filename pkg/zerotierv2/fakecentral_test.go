package zerotierv2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const testAPIKey = "test-key"

// fakeCentral is an in-memory Central V2 covering the operations the provider
// uses. It mirrors the server behaviors the provider depends on, as observed
// against a local Central V2: request bodies are decoded strictly, a create
// takes all config or none, updates merge over the current config, making a
// network public needs the publicNetworks entitlement (402 otherwise), reading
// a deleted object returns 403 (its permission tuples are gone), and reading a
// removed member returns 400 "unknown network member" (code 4101).
//
// It also assigns an authorized member an address straight away. The real
// controller does that once the device joins; doing it at once checks that the
// provider copes with addresses Central assigns.
type fakeCentral struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	nextID   int
	groups   map[string]*fakeGroup
	networks map[string]*fakeNetwork
	members  map[string]map[string]*fakeMember // network ID -> device ID
	accounts map[string]*fakeAccount
	keys     map[string]*fakeKey

	// publicNetworks is the org's entitlement to make a network public
	// (private = false). Central answers 402 without it.
	publicNetworks bool
}

type fakeGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OrgID       string `json:"orgId"`
}

type fakeNetwork struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	NetworkGroupID string        `json:"networkGroupId"`
	Kind           string        `json:"kind"`
	CreationTime   string        `json:"creationTime"`
	LastModified   string        `json:"lastModified"`
	Config         fakeNetConfig `json:"config"`
}

type fakeNetConfig struct {
	DNS                 fakeDNS         `json:"dns"`
	EnableBroadcast     bool            `json:"enableBroadcast"`
	V4Subnet            string          `json:"v4Subnet"`
	V6Subnet            string          `json:"v6Subnet"`
	V4IpAssignmentPools []fakeRange     `json:"v4IpAssignmentPools"`
	V6IpAssignmentPools []fakeRange     `json:"v6IpAssignmentPools"`
	MTU                 int             `json:"mtu"`
	MulticastLimit      int             `json:"multicastLimit"`
	Private             bool            `json:"private"`
	Routes              []fakeRoute     `json:"routes"`
	V4AssignmentMode    map[string]bool `json:"v4AssignmentMode"`
	V6AssignmentMode    map[string]bool `json:"v6AssignmentMode"`
}

type fakeDNS struct {
	Domain  string   `json:"domain"`
	Servers []string `json:"servers"`
}

type fakeRange struct {
	Start string `json:"ipRangeStart"`
	End   string `json:"ipRangeEnd"`
}

type fakeRoute struct {
	Target string `json:"target"`
	Via    string `json:"via"`
}

type fakeMember struct {
	DeviceID        string   `json:"deviceId"`
	NetworkID       string   `json:"networkId"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	ActiveBridge    bool     `json:"activeBridge"`
	NoAutoAssignIPs bool     `json:"noAutoAssignIps"`
	IPv4            []string `json:"ipv4Assignments"`
	IPv6            []string `json:"ipv6Assignments"`
	Identity        string   `json:"identity"`
	CreationTime    string   `json:"creationTime"`
	LastModified    string   `json:"lastModified"`
}

type fakeAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	OrgID        string `json:"orgId"`
	APIKeys      []any  `json:"apiKeys"`
	CreationTime string `json:"creationTime"`
	LastModified string `json:"lastModified"`
}

type fakeKey struct {
	ID               string `json:"id"`
	ServiceAccountID string `json:"serviceAccountId"`
	Description      string `json:"description"`
	Expires          string `json:"expires"`
	CreationTime     string `json:"creationTime"`
	LastModified     string `json:"lastModified"`
	Key              string `json:"key,omitempty"`
}

const fakeTime = "2026-11-03T12:00:00Z"

func newFakeCentral(t *testing.T) *fakeCentral {
	f := &fakeCentral{
		t:        t,
		groups:   map[string]*fakeGroup{},
		networks: map[string]*fakeNetwork{},
		members:  map[string]map[string]*fakeMember{},
		accounts: map[string]*fakeAccount{},
		keys:     map[string]*fakeKey{},

		publicNetworks: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/org/{org}/network-group", f.createGroup)
	mux.HandleFunc("GET /api/v2/network-group/{id}", f.getGroup)
	mux.HandleFunc("POST /api/v2/network-group/{id}", f.updateGroup)
	mux.HandleFunc("DELETE /api/v2/network-group/{id}", f.deleteGroup)
	mux.HandleFunc("POST /api/v2/network-group/{id}/network", f.createNetwork)
	mux.HandleFunc("GET /api/v2/network/{id}", f.getNetwork)
	mux.HandleFunc("POST /api/v2/network/{id}", f.updateNetwork)
	mux.HandleFunc("DELETE /api/v2/network/{id}", f.deleteNetwork)
	mux.HandleFunc("GET /api/v2/network/{id}/member/{dev}", f.getMember)
	mux.HandleFunc("POST /api/v2/network/{id}/member/{dev}", f.upsertMember)
	mux.HandleFunc("POST /api/v2/network/{id}/member/{dev}/{action}", f.memberStatus)
	mux.HandleFunc("DELETE /api/v2/network/{id}/member", f.deleteMembers)
	mux.HandleFunc("POST /api/v2/org/{org}/service-account", f.createAccount)
	mux.HandleFunc("GET /api/v2/service-account/{id}", f.getAccount)
	mux.HandleFunc("POST /api/v2/service-account/{id}", f.updateAccount)
	mux.HandleFunc("DELETE /api/v2/service-account/{id}", f.deleteAccount)
	mux.HandleFunc("POST /api/v2/service-account/{id}/api-key", f.createKey)
	mux.HandleFunc("GET /api/v2/api-key/{id}", f.getKey)
	mux.HandleFunc("POST /api/v2/api-key/{id}", f.updateKey)
	mux.HandleFunc("DELETE /api/v2/api-key/{id}", f.deleteKey)
	mux.HandleFunc("GET /api/v2/org/{org}/subscription", f.getSubscription)

	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch auth := r.Header.Get("Authorization"); {
		case auth == "Bearer "+testAPIKey:
			mux.ServeHTTP(w, r)
		case f.issuedKey(strings.TrimPrefix(auth, "Bearer ")):
			// Keys created through the API belong to service accounts with
			// no roles: authenticated, but allowed nothing.
			forbidden(w)
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "invalid or revoked api key"})
		}
	}))
	t.Cleanup(f.srv.Close)

	return f
}

func (f *fakeCentral) issuedKey(secret string) bool {
	_, ok := f.keys[strings.TrimPrefix(secret, "secret-")]
	return ok && strings.HasPrefix(secret, "secret-")
}

func (f *fakeCentral) id(n int) string {
	f.nextID++
	return fmt.Sprintf("%0*x", n, f.nextID)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func forbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"message": "403 Forbidden"})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "message": msg})
}

// decode reads the body strictly, like central-v2's httputils.NewStrictDecoder.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		badRequest(w, "invalid request body: "+err.Error())
		return false
	}
	return true
}

func (f *fakeCentral) createGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	g := &fakeGroup{ID: f.id(22), Name: req.Name, Description: deref(req.Description), OrgID: r.PathValue("org")}
	f.groups[g.ID] = g
	writeJSON(w, http.StatusOK, g)
}

func (f *fakeCentral) getGroup(w http.ResponseWriter, r *http.Request) {
	g, ok := f.groups[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (f *fakeCentral) updateGroup(w http.ResponseWriter, r *http.Request) {
	g, ok := f.groups[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Name != nil {
		g.Name = *req.Name
	}
	if req.Description != nil {
		g.Description = *req.Description
	}
	writeJSON(w, http.StatusOK, g)
}

func (f *fakeCentral) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := f.groups[id]; !ok {
		forbidden(w)
		return
	}
	delete(f.groups, id)
	for nid, n := range f.networks {
		if n.NetworkGroupID == id {
			delete(f.networks, nid)
			delete(f.members, nid)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "network group deleted"})
}

func (f *fakeCentral) createNetwork(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")
	if _, ok := f.groups[groupID]; !ok {
		forbidden(w)
		return
	}
	var req struct {
		Name        string          `json:"name"`
		Description *string         `json:"description"`
		Config      json.RawMessage `json:"config"`
	}
	if !decode(w, r, &req) {
		return
	}
	if len(req.Config) > 0 {
		// The provider never sends a create config: Central requires every
		// field when one is present.
		badRequest(w, "fake: config on create is not supported")
		return
	}
	n := &fakeNetwork{
		ID: f.id(16), Name: req.Name, Description: deref(req.Description), NetworkGroupID: groupID,
		Kind: "v1", CreationTime: fakeTime, LastModified: fakeTime,
		Config: fakeNetConfig{
			DNS:                 fakeDNS{Servers: []string{}},
			EnableBroadcast:     true,
			V4Subnet:            "10.147.20.0/24",
			V4IpAssignmentPools: []fakeRange{{Start: "10.147.20.1", End: "10.147.20.254"}},
			V6IpAssignmentPools: []fakeRange{},
			MTU:                 2800,
			MulticastLimit:      32,
			Private:             true,
			Routes:              []fakeRoute{},
			V4AssignmentMode:    map[string]bool{"zt": true},
			V6AssignmentMode:    map[string]bool{"zt": false, "6plane": false, "rfc4193": false},
		},
	}
	f.networks[n.ID] = n
	f.members[n.ID] = map[string]*fakeMember{}
	writeJSON(w, http.StatusOK, n)
}

func (f *fakeCentral) getNetwork(w http.ResponseWriter, r *http.Request) {
	n, ok := f.networks[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (f *fakeCentral) updateNetwork(w http.ResponseWriter, r *http.Request) {
	n, ok := f.networks[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	body, _ := readAll(r)

	var req struct {
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		Config      json.RawMessage `json:"config"`
	}
	if err := strictUnmarshal(body, &req); err != nil {
		badRequest(w, err.Error())
		return
	}

	if len(req.Config) > 0 {
		// Validate the shape Central's validator enforces before merging.
		var check struct {
			DNS *struct {
				Domain  string    `json:"domain"`
				Servers *[]string `json:"servers"`
			} `json:"dns"`
			Routes []fakeRoute `json:"routes"`
		}
		_ = json.Unmarshal(req.Config, &check)
		if check.DNS != nil && check.DNS.Servers == nil {
			badRequest(w, "dns.servers is required")
			return
		}
		for _, rt := range check.Routes {
			if rt.Via == "" {
				badRequest(w, "routes.via is required")
				return
			}
		}

		merged := n.Config
		if err := strictUnmarshal(req.Config, &merged); err != nil {
			badRequest(w, err.Error())
			return
		}
		if !merged.EnableBroadcast && merged.MulticastLimit != 0 {
			badRequest(w, "multicastLimit must be 0 when enableBroadcast is false")
			return
		}
		if n.Config.Private && !merged.Private && !f.publicNetworks {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"message": "402 Payment Required"})
			return
		}
		n.Config = merged
	}
	if req.Name != nil {
		n.Name = *req.Name
	}
	if req.Description != nil {
		n.Description = *req.Description
	}
	writeJSON(w, http.StatusOK, n)
}

func (f *fakeCentral) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := f.networks[id]; !ok {
		forbidden(w)
		return
	}
	delete(f.networks, id)
	delete(f.members, id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "network deleted"})
}

func (f *fakeCentral) member(w http.ResponseWriter, r *http.Request) (*fakeNetwork, *fakeMember, bool) {
	n, ok := f.networks[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return nil, nil, false
	}
	return n, f.members[n.ID][r.PathValue("dev")], true
}

func (f *fakeCentral) getMember(w http.ResponseWriter, r *http.Request) {
	_, m, ok := f.member(w, r)
	if !ok {
		return
	}
	if m == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 4101, "message": "unknown network member"})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (f *fakeCentral) upsertMember(w http.ResponseWriter, r *http.Request) {
	n, m, ok := f.member(w, r)
	if !ok {
		return
	}
	var req struct {
		Name            *string   `json:"name"`
		Description     *string   `json:"description"`
		ActiveBridge    *bool     `json:"activeBridge"`
		NoAutoAssignIPs *bool     `json:"noAutoAssignIps"`
		IPv4            *[]string `json:"ipv4Assignments"`
		IPv6            *[]string `json:"ipv6Assignments"`
	}
	if !decode(w, r, &req) {
		return
	}
	if m == nil {
		m = &fakeMember{
			DeviceID: r.PathValue("dev"), NetworkID: n.ID, Status: "not_authorized",
			IPv4: []string{}, IPv6: []string{}, CreationTime: fakeTime, LastModified: fakeTime,
		}
		f.members[n.ID][m.DeviceID] = m
	}
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Description != nil {
		m.Description = *req.Description
	}
	if req.ActiveBridge != nil {
		m.ActiveBridge = *req.ActiveBridge
	}
	if req.NoAutoAssignIPs != nil {
		m.NoAutoAssignIPs = *req.NoAutoAssignIPs
	}
	if req.IPv4 != nil {
		m.IPv4 = *req.IPv4
	}
	if req.IPv6 != nil {
		m.IPv6 = *req.IPv6
	}
	f.autoAssign(n, m)
	writeJSON(w, http.StatusOK, m)
}

// autoAssign gives an authorized member with no address the first pool
// address, the way the controller does.
func (f *fakeCentral) autoAssign(n *fakeNetwork, m *fakeMember) {
	if m.Status == "authorized" && !m.NoAutoAssignIPs && len(m.IPv4) == 0 && len(n.Config.V4IpAssignmentPools) > 0 {
		m.IPv4 = []string{n.Config.V4IpAssignmentPools[0].Start}
	}
}

func (f *fakeCentral) memberStatus(w http.ResponseWriter, r *http.Request) {
	n, m, ok := f.member(w, r)
	if !ok {
		return
	}
	if m == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 4101, "message": "unknown network member"})
		return
	}
	switch r.PathValue("action") {
	case "authorize":
		m.Status = "authorized"
	case "de-authorize":
		m.Status = "not_authorized"
	case "reject":
		m.Status = "rejected"
	default:
		http.NotFound(w, r)
		return
	}
	f.autoAssign(n, m)
	writeJSON(w, http.StatusOK, m)
}

func (f *fakeCentral) deleteMembers(w http.ResponseWriter, r *http.Request) {
	n, ok := f.networks[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	var req []struct {
		DeviceID string `json:"deviceId"`
	}
	if !decode(w, r, &req) {
		return
	}
	for _, d := range req {
		delete(f.members[n.ID], d.DeviceID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "members deleted"})
}

func (f *fakeCentral) createAccount(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("org")
	var req struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if !strings.HasSuffix(req.ID, "@org-"+org+".ztsa") {
		badRequest(w, "id must be <slug>@org-"+org+".ztsa")
		return
	}
	a := &fakeAccount{ID: req.ID, Name: req.Name, Description: deref(req.Description), OrgID: org,
		APIKeys: []any{}, CreationTime: fakeTime, LastModified: fakeTime}
	f.accounts[a.ID] = a
	writeJSON(w, http.StatusOK, a)
}

func (f *fakeCentral) getAccount(w http.ResponseWriter, r *http.Request) {
	a, ok := f.accounts[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (f *fakeCentral) updateAccount(w http.ResponseWriter, r *http.Request) {
	a, ok := f.accounts[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Description != nil {
		a.Description = *req.Description
	}
	writeJSON(w, http.StatusOK, a)
}

func (f *fakeCentral) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := f.accounts[id]; !ok {
		forbidden(w)
		return
	}
	delete(f.accounts, id)
	for kid, k := range f.keys {
		if k.ServiceAccountID == id {
			delete(f.keys, kid)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "service account deleted"})
}

func (f *fakeCentral) createKey(w http.ResponseWriter, r *http.Request) {
	sa := r.PathValue("id")
	if _, ok := f.accounts[sa]; !ok {
		forbidden(w)
		return
	}
	var req struct {
		Description *string `json:"description"`
		Expires     string  `json:"expires"`
	}
	if !decode(w, r, &req) {
		return
	}
	exp, err := time.Parse(time.RFC3339, req.Expires)
	if err != nil {
		badRequest(w, "expires: "+err.Error())
		return
	}
	k := &fakeKey{ID: f.id(22), ServiceAccountID: sa, Description: deref(req.Description),
		Expires: exp.UTC().Format(time.RFC3339Nano), CreationTime: fakeTime, LastModified: fakeTime}
	f.keys[k.ID] = k
	created := *k
	created.Key = "secret-" + k.ID
	writeJSON(w, http.StatusOK, created)
}

func (f *fakeCentral) getKey(w http.ResponseWriter, r *http.Request) {
	k, ok := f.keys[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	writeJSON(w, http.StatusOK, k)
}

func (f *fakeCentral) updateKey(w http.ResponseWriter, r *http.Request) {
	k, ok := f.keys[r.PathValue("id")]
	if !ok {
		forbidden(w)
		return
	}
	var req struct {
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Description != nil {
		k.Description = *req.Description
	}
	writeJSON(w, http.StatusOK, k)
}

func (f *fakeCentral) deleteKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := f.keys[id]; !ok {
		forbidden(w)
		return
	}
	delete(f.keys, id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "api key deleted"})
}

func (f *fakeCentral) getSubscription(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"entitlements": map[string]any{
			"publicNetworks": map[string]any{"value": f.publicNetworks, "kind": "boolean"},
		},
	})
}

func readAll(r *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

func strictUnmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// lock runs fn with the fake's state locked, for tests that change it
// behind Terraform's back.
func (f *fakeCentral) lock(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}
