package zerotierv2

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
	"github.com/zerotier/terraform-provider-zerotier/pkg/providerschema"
)

// baseResource carries the Central V2 client into each resource.
type baseResource struct {
	client *centralv2.ClientWithResponses
}

func (r *baseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *providerData, got %T", req.ProviderData))
		return
	}

	r.client = data.client
}

// api returns the client, or records an error if no V2 API key is configured.
func (r *baseResource) api(diags *diag.Diagnostics) *centralv2.ClientWithResponses {
	if r.client == nil {
		diags.AddError(
			"Central V2 API key not configured",
			fmt.Sprintf("%s must be set in the provider block, or ZEROTIER_CENTRAL_V2_API_KEY in the environment, to use zerotier_v2_* resources.", providerschema.CentralV2APIKey),
		)
	}
	return r.client
}

// removeIfGone drops a resource deleted outside Terraform from state and
// reports whether it did. See isGone for why 403 counts as gone.
func removeIfGone(ctx context.Context, err error, kind, id string, resp *resource.ReadResponse) bool {
	if !isGone(err) {
		return false
	}

	resp.Diagnostics.AddWarning(
		fmt.Sprintf("%s %s no longer found", kind, id),
		fmt.Sprintf("Central V2 returned %q when reading it, so it is assumed to have been deleted outside Terraform and will be planned for creation. If the API key has lost access instead, creating it again will fail with the same error.", err),
	)
	resp.State.RemoveResource(ctx)
	return true
}

func timeString(t time.Time) types.String {
	if t.IsZero() {
		return types.StringValue("")
	}
	return types.StringValue(t.UTC().Format(time.RFC3339))
}

func ptr[T any](v T) *T {
	return &v
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
