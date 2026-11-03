// Package zerotierv2 implements the Central V2 resources (zerotier_v2_*) on
// the Terraform plugin framework. It is served alongside the Central V1
// resources in pkg/zerotier through terraform-plugin-mux.
package zerotierv2

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
	"github.com/zerotier/terraform-provider-zerotier/pkg/providerschema"
)

// DefaultURL is the Central V2 API endpoint used when none is configured.
const DefaultURL = "https://central.zerotier.com"

type zerotierProvider struct {
	version string
}

type providerModel struct {
	CentralURL      types.String `tfsdk:"zerotier_central_url"`
	CentralToken    types.String `tfsdk:"zerotier_central_token"`
	CentralV2URL    types.String `tfsdk:"zerotier_central_v2_url"`
	CentralV2APIKey types.String `tfsdk:"zerotier_central_v2_api_key"`
}

// providerData is handed to every resource. client is nil when no V2 API key
// is configured, so a V1-only configuration still works.
type providerData struct {
	client *centralv2.ClientWithResponses
}

// New returns the Central V2 half of the provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &zerotierProvider{version: version}
	}
}

func (p *zerotierProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zerotier"
	resp.Version = p.version
}

// Schema must match the SDKv2 provider schema in pkg/zerotier exactly.
func (p *zerotierProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			providerschema.CentralURL: schema.StringAttribute{
				Optional:    true,
				Description: providerschema.CentralURLDescription,
			},
			providerschema.CentralToken: schema.StringAttribute{
				Optional:    true,
				Description: providerschema.CentralTokenDescription,
			},
			providerschema.CentralV2URL: schema.StringAttribute{
				Optional:    true,
				Description: providerschema.CentralV2URLDescription,
			},
			providerschema.CentralV2APIKey: schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: providerschema.CentralV2APIKeyDescription,
			},
		},
	}
}

func (p *zerotierProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := stringOrEnv(cfg.CentralV2URL, "ZEROTIER_CENTRAL_V2_URL", DefaultURL)
	apiKey := stringOrEnv(cfg.CentralV2APIKey, "ZEROTIER_CENTRAL_V2_API_KEY", "")

	data := &providerData{}
	if apiKey != "" {
		client, err := newClient(url, apiKey, fmt.Sprintf("terraform-provider-zerotier/%s", p.version), nil)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create Central V2 client", err.Error())
			return
		}
		data.client = client
	}

	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *zerotierProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newNetworkGroupResource,
		newNetworkResource,
		newMemberResource,
		newServiceAccountResource,
		newAPIKeyResource,
	}
}

func (p *zerotierProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func stringOrEnv(v types.String, env, def string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	if e := os.Getenv(env); e != "" {
		return e
	}
	return def
}

// newClient builds a Central V2 client that authenticates with a service
// account API key. httpClient may be nil to use http.DefaultClient.
func newClient(url, apiKey, userAgent string, httpClient *http.Client) (*centralv2.ClientWithResponses, error) {
	opts := []centralv2.ClientOption{
		centralv2.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("User-Agent", userAgent)
			return nil
		}),
	}
	if httpClient != nil {
		opts = append(opts, centralv2.WithHTTPClient(httpClient))
	}

	return centralv2.NewClientWithResponses(url, opts...)
}
