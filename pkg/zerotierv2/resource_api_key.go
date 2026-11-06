package zerotierv2

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
)

var (
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

type apiKeyResource struct {
	baseResource
}

type apiKeyModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Description      types.String `tfsdk:"description"`
	Expires          types.String `tfsdk:"expires"`
	Key              types.String `tfsdk:"key"`
	CreationTime     types.String `tfsdk:"creation_time"`
}

func newAPIKeyResource() resource.Resource {
	return &apiKeyResource{}
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An API key for a Central V2 service account. Central returns the key only when it is created, " +
			"so it is kept in Terraform state; treat the state as secret. Change `expires` (or use `-replace`) to rotate it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "API key ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_account_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the service account the key authenticates as.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description (up to 255 characters).",
				Validators:  []validator.String{stringvalidator.LengthAtMost(255)},
			},
			"expires": schema.StringAttribute{
				Required:      true,
				Description:   "When the key expires (RFC 3339, in the future). Changing it creates a new key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{rfc3339Validator{}},
			},
			"key": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The API key. Only available for keys created by Terraform, not imported ones.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"creation_time": schema.StringAttribute{
				Computed:      true,
				Description:   "When the key was created (RFC 3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	expires, err := time.Parse(time.RFC3339, plan.Expires.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("expires"), "Invalid expires", err.Error())
		return
	}

	res, err := api.CreateApiKeyWithResponse(ctx, plan.ServiceAccountID.ValueString(), centralv2.CreateApiKeyRequest{
		Description: ptr(plan.Description.ValueString()),
		Expires:     expires,
	})
	key, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create API key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiKeyModel{
		ID:               types.StringValue(key.Id),
		ServiceAccountID: types.StringValue(key.ServiceAccountId),
		Description:      types.StringValue(key.Description),
		Expires:          expiresValue(plan.Expires, key.Expires),
		Key:              types.StringValue(key.Key),
		CreationTime:     timeString(key.CreationTime),
	})...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.GetApiKeyWithResponse(ctx, state.ID.ValueString())
	key, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		if removeIfGone(ctx, err, "API key", state.ID.ValueString(), resp) {
			return
		}
		resp.Diagnostics.AddError("Unable to read API key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiKeyFromResponse(key, state))...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.UpdateApiKeyWithResponse(ctx, state.ID.ValueString(), centralv2.UpdateApiKeyRequest{
		Description: ptr(plan.Description.ValueString()),
	})
	key, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update API key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiKeyFromResponse(key, state))...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.DeleteApiKeyWithResponse(ctx, state.ID.ValueString())
	if err := checkNoBody(err, deref(res).HTTPResponse, deref(res).Body); err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Unable to delete API key", err.Error())
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// apiKeyFromResponse keeps the key from state: Central never returns it again.
func apiKeyFromResponse(key *centralv2.ApiKeyResponse, prior apiKeyModel) apiKeyModel {
	m := apiKeyModel{
		ID:               types.StringValue(key.Id),
		ServiceAccountID: types.StringValue(key.ServiceAccountId),
		Description:      types.StringValue(key.Description),
		Expires:          prior.Expires,
		Key:              prior.Key,
		CreationTime:     timeString(key.CreationTime),
	}
	if key.Expires != nil {
		m.Expires = expiresValue(prior.Expires, *key.Expires)
	}
	return m
}

// expiresValue keeps the configured spelling of expires when it denotes the
// same instant Central returned, so a timezone offset or precision difference
// does not show as a change.
func expiresValue(configured types.String, actual time.Time) types.String {
	if known(configured) {
		if t, err := time.Parse(time.RFC3339, configured.ValueString()); err == nil && t.Equal(actual) {
			return configured
		}
	}
	return types.StringValue(actual.UTC().Format(time.RFC3339))
}

type rfc3339Validator struct{}

func (rfc3339Validator) Description(context.Context) string {
	return "value must be an RFC 3339 timestamp"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (rfc3339Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !known(req.ConfigValue) {
		return
	}
	if _, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid timestamp",
			fmt.Sprintf("%q is not an RFC 3339 timestamp, for example 2027-01-01T00:00:00Z.", req.ConfigValue.ValueString()))
	}
}
