package zerotierv2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
)

// Acceptance tests against a real Central V2. They run only with TF_ACC=1 and:
//
//	ZEROTIER_CENTRAL_V2_URL      API endpoint, e.g. http://localhost:8085
//	ZEROTIER_CENTRAL_V2_API_KEY  service account API key with org Admin
//	ZEROTIER_CENTRAL_V2_ORG_ID   org to create test objects in
//
// The provider block is empty, so the provider's environment variables are
// exercised too. Everything the tests create is destroyed at the end.

type accEnv struct {
	url, apiKey, orgID string
}

func accEnvSet() bool {
	return os.Getenv("TF_ACC") != ""
}

func accPreCheck(t *testing.T) accEnv {
	t.Helper()
	env := accEnv{
		url:    os.Getenv("ZEROTIER_CENTRAL_V2_URL"),
		apiKey: os.Getenv("ZEROTIER_CENTRAL_V2_API_KEY"),
		orgID:  os.Getenv("ZEROTIER_CENTRAL_V2_ORG_ID"),
	}
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests against Central V2")
	}
	if env.url == "" || env.apiKey == "" || env.orgID == "" {
		t.Fatal("ZEROTIER_CENTRAL_V2_URL, ZEROTIER_CENTRAL_V2_API_KEY and ZEROTIER_CENTRAL_V2_ORG_ID must be set")
	}
	return env
}

// client returns a Central V2 client with the given key, for checks made
// outside Terraform.
func (e accEnv) client(t *testing.T, apiKey string) *centralv2.ClientWithResponses {
	t.Helper()
	c, err := newClient(e.url, apiKey, "terraform-provider-zerotier/acctest", nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// entitlement reports a boolean entitlement of the test org's plan. The
// subscription endpoint is not in the generated client, so it is read
// directly.
func (e accEnv) entitlement(t *testing.T, name string) bool {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.url+"/api/v2/org/"+e.orgID+"/subscription", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var sub struct {
		// Numeric entitlements (device and network limits) sit alongside the
		// boolean ones, so values are decoded loosely.
		Entitlements map[string]struct {
			Value any `json:"value"`
		} `json:"entitlements"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&sub) != nil {
		t.Fatalf("reading org %s subscription: %s", e.orgID, resp.Status)
	}
	return sub.Entitlements[name].Value == true
}

func accName(prefix string) string {
	return prefix + "-" + acctest.RandString(8)
}

// accDeviceID returns a random ZeroTier address. Addresses starting with ff
// are reserved.
func accDeviceID(t *testing.T) string {
	b := make([]byte, 5)
	for {
		if _, err := rand.Read(b); err != nil {
			t.Fatal(err)
		}
		if b[0] != 0xff && b[0] != 0 {
			return hex.EncodeToString(b)
		}
	}
}

const accProvider = "provider \"zerotier\" {}\n"

func accGroup(orgID, name string) string {
	return fmt.Sprintf(`
resource "zerotier_v2_network_group" "g" {
  org_id = %q
  name   = %q
}
`, orgID, name)
}

// accCheckGone verifies with a direct GET that every resource of the type is
// gone after destroy, and logs the status Central answered with.
func accCheckGone(t *testing.T, env accEnv, resourceType string, get func(c *centralv2.ClientWithResponses, rs *terraform.ResourceState) (int, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := env.client(t, env.apiKey)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			status, err := get(c, rs)
			if err != nil {
				return err
			}
			t.Logf("after destroy: GET %s %s -> %d", resourceType, rs.Primary.ID, status)
			if status != http.StatusNotFound && status != http.StatusForbidden {
				return fmt.Errorf("%s %s still exists (GET returned %d)", resourceType, rs.Primary.ID, status)
			}
		}
		return nil
	}
}

func TestAccNetworkGroupAndNetwork(t *testing.T) {
	env := accPreCheck(t)
	group := accGroup(env.orgID, accName("tf-acc"))
	name := accName("tf-acc")

	network := func(body string) string {
		return accProvider + group + fmt.Sprintf(`
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = %q
%s
}
`, name, body)
	}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			accCheckGone(t, env, "zerotier_v2_network", func(c *centralv2.ClientWithResponses, rs *terraform.ResourceState) (int, error) {
				res, err := c.GetNetworkWithResponse(context.Background(), rs.Primary.ID)
				if err != nil {
					return 0, err
				}
				return res.StatusCode(), nil
			}),
			accCheckGone(t, env, "zerotier_v2_network_group", func(c *centralv2.ClientWithResponses, rs *terraform.ResourceState) (int, error) {
				res, err := c.GetNetworkGroupWithResponse(context.Background(), rs.Primary.ID)
				if err != nil {
					return 0, err
				}
				return res.StatusCode(), nil
			}),
		),
		Steps: []resource.TestStep{
			{
				// Minimal: Central picks the subnet and pool.
				Config: network(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network_group.g", "org_id", env.orgID),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "kind", "v1"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "2800"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "multicast_limit", "32"),
					resource.TestCheckResourceAttrSet("zerotier_v2_network.n", "v4_subnet"),
					resource.TestCheckResourceAttrSet("zerotier_v2_network.n", "v4_assignment_pools.0.start"),
					resource.TestCheckResourceAttrSet("zerotier_v2_network.n", "creation_time"),
				),
			},
			{
				// Every setting, applied to the existing network.
				Config: network(`
  description         = "terraform acceptance test"
  mtu                 = 1400
  multicast_limit     = 64
  v4_subnet           = "10.123.45.0/24"
  v4_assignment_pools = [{ start = "10.123.45.10", end = "10.123.45.250" }]
  routes              = [{ target = "192.168.123.0/24", via = "10.123.45.1" }]
  dns_domain          = "acc.example.com"
  dns_servers         = ["10.123.45.53"]
  v6_assign_rfc4193   = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "1400"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "multicast_limit", "64"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v4_subnet", "10.123.45.0/24"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "routes.0.via", "10.123.45.1"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_servers.0", "10.123.45.53"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v6_assign_rfc4193", "true"),
				),
			},
			{
				// Removing settings returns them to the defaults.
				Config: network(`
  v4_subnet           = "10.123.45.0/24"
  v4_assignment_pools = [{ start = "10.123.45.10", end = "10.123.45.250" }]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "routes.#", "0"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_servers.#", "0"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "2800"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "private", "true"),
				),
			},
			{ResourceName: "zerotier_v2_network.n", ImportState: true, ImportStateVerify: true},
			{ResourceName: "zerotier_v2_network_group.g", ImportState: true, ImportStateVerify: true},
		},
	})
}

// Making a network public (private = false) needs the org's publicNetworks
// entitlement. With it the change applies; without it Central answers 402 and
// the provider explains that the plan lacks the setting.
func TestAccNetworkPublic(t *testing.T) {
	env := accPreCheck(t)
	entitled := env.entitlement(t, "publicNetworks")
	t.Logf("org %s publicNetworks entitlement: %v", env.orgID, entitled)

	config := accProvider + accGroup(env.orgID, accName("tf-acc")) + fmt.Sprintf(`
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = %q
  private          = false
}
`, accName("tf-acc"))

	step := resource.TestStep{Config: config}
	if entitled {
		step.Check = resource.TestCheckResourceAttr("zerotier_v2_network.n", "private", "false")
	} else {
		step.ExpectError = regexp.MustCompile(`(?s)402.*plan\s+doesn't\s+allow\s+this\s+change`)
	}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps:                    []resource.TestStep{step},
	})
}

// Deleting a network outside Terraform. Central V2 is expected to answer
// the next read with 403 (its permission tuples are gone); the provider must
// plan to re-create it either way.
func TestAccNetworkDeletedOutsideTerraform(t *testing.T) {
	env := accPreCheck(t)
	config := accProvider + accGroup(env.orgID, accName("tf-acc")) + fmt.Sprintf(`
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = %q
}
`, accName("tf-acc"))

	var networkID string

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					networkID = s.RootModule().Resources["zerotier_v2_network.n"].Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					c := env.client(t, env.apiKey)
					del, err := c.DeleteNetworkWithResponse(context.Background(), networkID)
					if err != nil || del.StatusCode() != http.StatusOK {
						t.Fatalf("deleting network %s outside Terraform: %v %s", networkID, err, del.Body)
					}
					get, err := c.GetNetworkWithResponse(context.Background(), networkID)
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("GET deleted network %s -> %d %s", networkID, get.StatusCode(), get.Body)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zerotier_v2_network.n", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccMember(t *testing.T) {
	env := accPreCheck(t)
	deviceID := accDeviceID(t)

	base := accProvider + accGroup(env.orgID, accName("tf-acc")) + fmt.Sprintf(`
resource "zerotier_v2_network" "n" {
  network_group_id    = zerotier_v2_network_group.g.id
  name                = %q
  v4_subnet           = "10.123.46.0/24"
  v4_assignment_pools = [{ start = "10.123.46.10", end = "10.123.46.250" }]
}
`, accName("tf-acc"))

	member := func(body string) string {
		return base + fmt.Sprintf(`
resource "zerotier_v2_member" "m" {
  network_id = zerotier_v2_network.n.id
  device_id  = %q
%s
}
`, deviceID, body)
	}

	var memberNetworkID string

	logMember := func(s *terraform.State) error {
		a := s.RootModule().Resources["zerotier_v2_member.m"].Primary.Attributes
		memberNetworkID = a["network_id"]
		t.Logf("member %s: status=%s ipv4=%s ipv4_assignments.#=%s", a["id"], a["status"], a["ipv4_assignments.0"], a["ipv4_assignments.#"])
		return nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		CheckDestroy: accCheckGone(t, env, "zerotier_v2_member", func(c *centralv2.ClientWithResponses, rs *terraform.ResourceState) (int, error) {
			res, err := c.GetNetworkMemberWithResponse(context.Background(), rs.Primary.Attributes["network_id"], rs.Primary.Attributes["device_id"])
			if err != nil {
				return 0, err
			}
			return res.StatusCode(), nil
		}),
		Steps: []resource.TestStep{
			{
				// A device that has never joined, pre-authorized.
				Config: member(`  name = "acc-device"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "authorized"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "name", "acc-device"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "device_id", deviceID),
					resource.TestMatchResourceAttr("zerotier_v2_member.m", "id", regexp.MustCompile(`^[0-9a-f]{16}/`+deviceID+`$`)),
					logMember,
				),
			},
			{
				Config: member(`
  name             = "acc-device"
  status           = "not_authorized"
  ipv4_assignments = ["10.123.46.77"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "not_authorized"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "ipv4_assignments.#", "1"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "ipv4_assignments.0", "10.123.46.77"),
					logMember,
				),
			},
			{
				Config: member(`
  name   = "acc-device"
  status = "rejected"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "rejected"),
					logMember,
				),
			},
			{ResourceName: "zerotier_v2_member.m", ImportState: true, ImportStateVerify: true},
			{
				// The dash form used by V1 member IDs.
				ResourceName: "zerotier_v2_member.m",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					a := s.RootModule().Resources["zerotier_v2_member.m"].Primary.Attributes
					return a["network_id"] + "-" + a["device_id"], nil
				},
				ImportStateVerify: true,
			},
			{
				// Removed outside Terraform: Central answers 400 "unknown network
				// member" (code 4101); the provider must plan a re-create.
				PreConfig: func() {
					c := env.client(t, env.apiKey)
					networkID := memberNetworkID
					del, err := c.RemoveNetworkMembersWithResponse(context.Background(), networkID,
						[]centralv2.DeleteMembersRequest{{DeviceId: deviceID}})
					if err != nil || del.StatusCode() != http.StatusOK {
						t.Fatalf("removing member outside Terraform: %v %s", err, del.Body)
					}
					get, err := c.GetNetworkMemberWithResponse(context.Background(), networkID, deviceID)
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("GET removed member %s/%s -> %d %s", networkID, deviceID, get.StatusCode(), get.Body)
				},
				Config: member(`
  name   = "acc-device"
  status = "rejected"
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zerotier_v2_member.m", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccServiceAccountAndAPIKey(t *testing.T) {
	env := accPreCheck(t)
	slug := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	config := func(saDescription, keyDescription string) string {
		return accProvider + fmt.Sprintf(`
resource "zerotier_v2_service_account" "sa" {
  org_id      = %q
  slug        = %q
  name        = "Terraform acceptance test"
  description = %q
}

resource "zerotier_v2_api_key" "k" {
  service_account_id = zerotier_v2_service_account.sa.id
  description        = %q
  expires            = "2030-01-01T00:00:00Z"
}
`, env.orgID, slug, saDescription, keyDescription)
	}

	var createdKey string

	// keyStatus calls the API with the created key. The new service account
	// has no roles, so a working key gets 403; a revoked one gets 401.
	keyStatus := func() int {
		res, err := env.client(t, createdKey).GetOrgWithResponse(context.Background(), env.orgID)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode()
	}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		CheckDestroy: func(s *terraform.State) error {
			status := keyStatus()
			t.Logf("after destroy: request with the deleted key -> %d", status)
			if status != http.StatusUnauthorized {
				return fmt.Errorf("deleted API key still authenticates (got %d, want 401)", status)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: config("created", "created"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_service_account.sa", "id", slug+"@org-"+env.orgID+".ztsa"),
					resource.TestCheckResourceAttr("zerotier_v2_service_account.sa", "slug", slug),
					resource.TestCheckResourceAttrSet("zerotier_v2_api_key.k", "key"),
					resource.TestCheckResourceAttr("zerotier_v2_api_key.k", "expires", "2030-01-01T00:00:00Z"),
					func(s *terraform.State) error {
						createdKey = s.RootModule().Resources["zerotier_v2_api_key.k"].Primary.Attributes["key"]
						status := keyStatus()
						t.Logf("request with the new key -> %d", status)
						if status == http.StatusUnauthorized {
							return fmt.Errorf("the API key Terraform created does not authenticate (401)")
						}
						return nil
					},
				),
			},
			{
				Config: config("updated", "updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zerotier_v2_service_account.sa", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("zerotier_v2_api_key.k", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_service_account.sa", "description", "updated"),
					resource.TestCheckResourceAttr("zerotier_v2_api_key.k", "description", "updated"),
					resource.TestMatchResourceAttr("zerotier_v2_api_key.k", "key", regexp.MustCompile(`.+`)),
				),
			},
			{ResourceName: "zerotier_v2_service_account.sa", ImportState: true, ImportStateVerify: true},
			{
				ResourceName:            "zerotier_v2_api_key.k",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}
