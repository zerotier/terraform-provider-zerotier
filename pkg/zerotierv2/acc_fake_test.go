package zerotierv2

import "testing"

// TestAccHarnessAgainstFake runs the acceptance tests against the in-memory
// fake, so the harness itself is checked by every `go test`. It is skipped
// when TF_ACC is set, since the acceptance tests then run against the real
// Central V2 named in the environment.
func TestAccHarnessAgainstFake(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
	if accEnvSet() {
		t.Skip("TF_ACC is set; the acceptance tests run against the configured Central V2")
	}

	tests := []struct {
		name           string
		test           func(*testing.T)
		publicNetworks bool
	}{
		{"NetworkGroupAndNetwork", TestAccNetworkGroupAndNetwork, false},
		{"NetworkPublic/entitled", TestAccNetworkPublic, true},
		{"NetworkPublic/not entitled", TestAccNetworkPublic, false},
		{"NetworkDeletedOutsideTerraform", TestAccNetworkDeletedOutsideTerraform, false},
		{"Member", TestAccMember, false},
		{"ServiceAccountAndAPIKey", TestAccServiceAccountAndAPIKey, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeCentral(t)
			f.publicNetworks = tc.publicNetworks
			t.Setenv("TF_ACC", "1")
			t.Setenv("ZEROTIER_CENTRAL_V2_URL", f.srv.URL)
			t.Setenv("ZEROTIER_CENTRAL_V2_API_KEY", testAPIKey)
			t.Setenv("ZEROTIER_CENTRAL_V2_ORG_ID", testOrgID)
			tc.test(t)
		})
	}
}
