package zerotierv2

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/zerotier/terraform-provider-zerotier/pkg/zerotier"
)

const testOrgID = "0123456789abcdefghijkl"

// providerFactories serves the same muxed provider as main.go, so these tests
// also check that the V1 and V2 provider schemas agree.
var providerFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"zerotier": func() (tfprotov5.ProviderServer, error) {
		mux, err := tf5muxserver.NewMuxServer(context.Background(),
			zerotier.Provider().GRPCProvider,
			providerserver.NewProtocol5(New("test")()),
		)
		if err != nil {
			return nil, err
		}
		return mux.ProviderServer(), nil
	},
}

func providerConfig(f *fakeCentral) string {
	return fmt.Sprintf(`
provider "zerotier" {
  zerotier_central_v2_url     = %q
  zerotier_central_v2_api_key = %q
}
`, f.srv.URL, testAPIKey)
}

func TestNetworkGroupAndNetwork(t *testing.T) {
	f := newFakeCentral(t)

	group := fmt.Sprintf(`
resource "zerotier_v2_network_group" "g" {
  org_id = %q
  name   = "lab"
}
`, testOrgID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Minimal network: Central picks the subnet and pool.
				Config: providerConfig(f) + group + `
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = "minimal"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network_group.g", "org_id", testOrgID),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "kind", "v1"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "2800"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v4_subnet", "10.147.20.0/24"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v4_assignment_pools.0.start", "10.147.20.1"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v4_auto_assign", "true"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_servers.#", "0"),
				),
			},
			{
				// Full config: every field reaches Central in one apply.
				Config: providerConfig(f) + group + `
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = "configured"
  description      = "managed by terraform"
  mtu              = 1400
  multicast_limit  = 64
  private          = false
  v4_subnet        = "10.99.0.0/24"
  v4_assignment_pools = [{ start = "10.99.0.10", end = "10.99.0.250" }]
  routes = [{ target = "192.168.50.0/24", via = "10.99.0.1" }]
  dns_domain        = "lab.example.com"
  dns_servers       = ["10.99.0.53"]
  v6_assign_rfc4193 = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "name", "configured"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "1400"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "multicast_limit", "64"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "private", "false"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v4_subnet", "10.99.0.0/24"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "routes.0.via", "10.99.0.1"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_servers.0", "10.99.0.53"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v6_assign_rfc4193", "true"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "v6_assign_6plane", "false"),
				),
			},
			{
				// Removing routes and DNS from config clears them in Central.
				Config: providerConfig(f) + group + `
resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = "configured"
  v4_subnet        = "10.99.0.0/24"
  v4_assignment_pools = [{ start = "10.99.0.10", end = "10.99.0.250" }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "routes.#", "0"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_domain", ""),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "dns_servers.#", "0"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "mtu", "2800"),
					resource.TestCheckResourceAttr("zerotier_v2_network.n", "description", ""),
				),
			},
			{
				ResourceName:      "zerotier_v2_network.n",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "zerotier_v2_network_group.g",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestNetworkBroadcastValidation(t *testing.T) {
	f := newFakeCentral(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: providerConfig(f) + `
resource "zerotier_v2_network" "n" {
  network_group_id = "g"
  name             = "nobroadcast"
  enable_broadcast = false
}
`,
			ExpectError: regexp.MustCompile(`multicast_limit = 0 when enable_broadcast is false`),
		}},
	})
}

func TestNetworkDeletedOutsideTerraform(t *testing.T) {
	f := newFakeCentral(t)

	config := providerConfig(f) + fmt.Sprintf(`
resource "zerotier_v2_network_group" "g" {
  org_id = %q
  name   = "lab"
}

resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = "doomed"
}
`, testOrgID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// Central answers 403 for a deleted network; it is re-created.
				PreConfig: func() {
					f.lock(func() {
						for id := range f.networks {
							delete(f.networks, id)
						}
					})
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

func TestMember(t *testing.T) {
	f := newFakeCentral(t)

	network := providerConfig(f) + fmt.Sprintf(`
resource "zerotier_v2_network_group" "g" {
  org_id = %q
  name   = "lab"
}

resource "zerotier_v2_network" "n" {
  network_group_id = zerotier_v2_network_group.g.id
  name             = "members"
}
`, testOrgID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Default status is authorized; Central auto-assigns an address.
				Config: network + `
resource "zerotier_v2_member" "m" {
  network_id = zerotier_v2_network.n.id
  device_id  = "abcdef0123"
  name       = "laptop"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "authorized"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "ipv4_assignments.0", "10.147.20.1"),
					resource.TestCheckResourceAttrPair("zerotier_v2_member.m", "network_id", "zerotier_v2_network.n", "id"),
				),
			},
			{
				Config: network + `
resource "zerotier_v2_member" "m" {
  network_id       = zerotier_v2_network.n.id
  device_id        = "abcdef0123"
  name             = "laptop"
  status           = "not_authorized"
  ipv4_assignments = ["10.147.20.77"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "not_authorized"),
					resource.TestCheckResourceAttr("zerotier_v2_member.m", "ipv4_assignments.0", "10.147.20.77"),
				),
			},
			{
				Config: network + `
resource "zerotier_v2_member" "m" {
  network_id = zerotier_v2_network.n.id
  device_id  = "abcdef0123"
  status     = "rejected"
}
`,
				Check: resource.TestCheckResourceAttr("zerotier_v2_member.m", "status", "rejected"),
			},
			{
				ResourceName:      "zerotier_v2_member.m",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// The dash form used by V1 member IDs also imports.
				ResourceName: "zerotier_v2_member.m",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["zerotier_v2_member.m"]
					return rs.Primary.Attributes["network_id"] + "-" + rs.Primary.Attributes["device_id"], nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestServiceAccountAndAPIKey(t *testing.T) {
	f := newFakeCentral(t)

	config := func(saDescription, keyDescription string) string {
		return providerConfig(f) + fmt.Sprintf(`
resource "zerotier_v2_service_account" "ci" {
  org_id      = %q
  slug        = "ci-bot"
  name        = "CI bot"
  description = %q
}

resource "zerotier_v2_api_key" "ci" {
  service_account_id = zerotier_v2_service_account.ci.id
  description        = %q
  expires            = "2030-01-01T00:00:00+01:00"
}
`, testOrgID, saDescription, keyDescription)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config("deploys", "github actions"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zerotier_v2_service_account.ci", "id", "ci-bot@org-"+testOrgID+".ztsa"),
					resource.TestCheckResourceAttr("zerotier_v2_service_account.ci", "slug", "ci-bot"),
					resource.TestMatchResourceAttr("zerotier_v2_api_key.ci", "key", regexp.MustCompile(`^secret-`)),
					// The configured offset is kept although Central returns UTC.
					resource.TestCheckResourceAttr("zerotier_v2_api_key.ci", "expires", "2030-01-01T00:00:00+01:00"),
				),
			},
			{
				// Descriptions update in place; the key is not replaced.
				Config: config("deploys prod", "github actions prod"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zerotier_v2_service_account.ci", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("zerotier_v2_api_key.ci", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestMatchResourceAttr("zerotier_v2_api_key.ci", "key", regexp.MustCompile(`^secret-`)),
			},
			{
				ResourceName:      "zerotier_v2_service_account.ci",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:            "zerotier_v2_api_key.ci",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key", "expires"},
			},
		},
	})
}

// The V1 token is optional now, so a V2-only configuration can still use
// zerotier_identity, and V1 resources report the missing token.
func TestV1WithoutToken(t *testing.T) {
	t.Setenv("ZEROTIER_CENTRAL_TOKEN", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "zerotier" {}

resource "zerotier_identity" "node" {}
`,
				Check: resource.TestCheckResourceAttrSet("zerotier_identity.node", "public_key"),
			},
			{
				Config: `
provider "zerotier" {}

resource "zerotier_network" "n" {
  name = "v1"
}
`,
				ExpectError: regexp.MustCompile(`zerotier_central_token must be specified`),
			},
		},
	})
}

func TestV2ResourceWithoutAPIKey(t *testing.T) {
	t.Setenv("ZEROTIER_CENTRAL_V2_API_KEY", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "zerotier" {}

resource "zerotier_v2_network_group" "g" {
  org_id = "org"
  name   = "lab"
}
`,
			ExpectError: regexp.MustCompile(`Central V2 API key not configured`),
		}},
	})
}
