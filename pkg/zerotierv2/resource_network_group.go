package zerotierv2

import (
	"context"

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
	_ resource.ResourceWithConfigure   = &networkGroupResource{}
	_ resource.ResourceWithImportState = &networkGroupResource{}
)

type networkGroupResource struct {
	baseResource
}

type networkGroupModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func newNetworkGroupResource() resource.Resource {
	return &networkGroupResource{}
}

func (r *networkGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_network_group"
}

func (r *networkGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Central V2 network group: a collection of networks within an organization. " +
			"Deleting a network group also deletes every network in it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Network group ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the organization the network group belongs to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the network group.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the network group.",
			},
		},
	}
}

func (r *networkGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.CreateNetworkGroupWithResponse(ctx, plan.OrgID.ValueString(), centralv2.CreateNetworkGroupRequest{
		Name:        plan.Name.ValueString(),
		Description: ptr(plan.Description.ValueString()),
	})
	group, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create network group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkGroupToModel(group))...)
}

func (r *networkGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.GetNetworkGroupWithResponse(ctx, state.ID.ValueString())
	group, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		if removeIfGone(ctx, err, "Network group", state.ID.ValueString(), resp) {
			return
		}
		resp.Diagnostics.AddError("Unable to read network group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkGroupToModel(group))...)
}

func (r *networkGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state networkGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.UpdateNetworkGroupWithResponse(ctx, state.ID.ValueString(), centralv2.UpdateNetworkGroupRequest{
		Name:        ptr(plan.Name.ValueString()),
		Description: ptr(plan.Description.ValueString()),
	})
	group, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update network group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkGroupToModel(group))...)
}

func (r *networkGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.DeleteNetworkGroupWithResponse(ctx, state.ID.ValueString())
	if err := checkNoBody(err, deref(res).HTTPResponse, deref(res).Body); err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Unable to delete network group", err.Error())
	}
}

func (r *networkGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func networkGroupToModel(g *centralv2.NetworkGroup) networkGroupModel {
	return networkGroupModel{
		ID:          types.StringValue(g.Id),
		OrgID:       types.StringValue(g.OrgId),
		Name:        types.StringValue(g.Name),
		Description: types.StringValue(g.Description),
	}
}
