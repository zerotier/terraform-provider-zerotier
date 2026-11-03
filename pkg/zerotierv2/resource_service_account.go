package zerotierv2

import (
	"context"
	"regexp"
	"strings"

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
	_ resource.ResourceWithConfigure   = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}
)

// Central V2 builds service account IDs as <slug>@org-<org id>.ztsa
// (internal/validation/sa_id_format.go in central-v2).
var serviceAccountSlugPattern = regexp.MustCompile(`^[^@]{2,32}$`)

type serviceAccountResource struct {
	baseResource
}

type serviceAccountModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Slug         types.String `tfsdk:"slug"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	CreationTime types.String `tfsdk:"creation_time"`
}

func newServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Central V2 service account, an identity for programmatic API access. " +
			"Create API keys for it with zerotier_v2_api_key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Service account ID, `<slug>@org-<org_id>.ztsa`. Also the principal used to grant it roles.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the organization the service account belongs to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"slug": schema.StringAttribute{
				Required:      true,
				Description:   "Short identifier that forms the first part of the ID (2 to 32 characters, no `@`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(serviceAccountSlugPattern, "must be 2 to 32 characters and must not contain @"),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name (3 to 100 characters).",
				Validators:  []validator.String{stringvalidator.LengthBetween(3, 100)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description (up to 255 characters).",
				Validators:  []validator.String{stringvalidator.LengthAtMost(255)},
			},
			"creation_time": schema.StringAttribute{
				Computed:      true,
				Description:   "When the service account was created (RFC 3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrgID.ValueString()
	res, err := api.CreateServiceAccountWithResponse(ctx, orgID, centralv2.CreateServiceAccountRequest{
		Id:          plan.Slug.ValueString() + "@org-" + orgID + ".ztsa",
		Name:        plan.Name.ValueString(),
		Description: ptr(plan.Description.ValueString()),
	})
	sa, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create service account", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceAccountToModel(sa))...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.GetServiceAccountWithResponse(ctx, state.ID.ValueString())
	sa, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		if removeIfGone(ctx, err, "Service account", state.ID.ValueString(), resp) {
			return
		}
		resp.Diagnostics.AddError("Unable to read service account", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceAccountToModel(sa))...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.UpdateServiceAccountWithResponse(ctx, state.ID.ValueString(), centralv2.UpdateServiceAccountRequest{
		Name:        ptr(plan.Name.ValueString()),
		Description: ptr(plan.Description.ValueString()),
	})
	sa, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update service account", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceAccountToModel(sa))...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.DeleteServiceAccountWithResponse(ctx, state.ID.ValueString())
	if err := checkNoBody(err, deref(res).HTTPResponse, deref(res).Body); err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Unable to delete service account", err.Error())
	}
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func serviceAccountToModel(sa *centralv2.ServiceAccountResponse) serviceAccountModel {
	slug, _, _ := strings.Cut(sa.Id, "@")

	m := serviceAccountModel{
		ID:           types.StringValue(sa.Id),
		OrgID:        types.StringValue(sa.OrgId),
		Slug:         types.StringValue(slug),
		Name:         types.StringValue(sa.Name),
		Description:  types.StringValue(sa.Description),
		CreationTime: types.StringValue(""),
	}
	if sa.CreationTime != nil {
		m.CreationTime = timeString(*sa.CreationTime)
	}
	return m
}
