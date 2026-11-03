package zerotierv2

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
)

var (
	_ resource.ResourceWithConfigure   = &memberResource{}
	_ resource.ResourceWithImportState = &memberResource{}
)

var (
	networkIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)
	deviceIDPattern  = regexp.MustCompile(`^[0-9a-f]{10}$`)
)

type memberResource struct {
	baseResource
}

type memberModel struct {
	ID                  types.String `tfsdk:"id"`
	NetworkID           types.String `tfsdk:"network_id"`
	DeviceID            types.String `tfsdk:"device_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Status              types.String `tfsdk:"status"`
	ActiveBridge        types.Bool   `tfsdk:"active_bridge"`
	NoAutoAssignIPs     types.Bool   `tfsdk:"no_auto_assign_ips"`
	IPv4Assignments     types.List   `tfsdk:"ipv4_assignments"`
	IPv6Assignments     types.List   `tfsdk:"ipv6_assignments"`
	Identity            types.String `tfsdk:"identity"`
	IPAssignment6plane  types.String `tfsdk:"ip_assignment_6plane"`
	IPAssignmentRFC4193 types.String `tfsdk:"ip_assignment_rfc4193"`
	CreationTime        types.String `tfsdk:"creation_time"`
}

func newMemberResource() resource.Resource {
	return &memberResource{}
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_member"
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A device's membership of a Central V2 network. The device does not need to have joined yet; " +
			"creating the member pre-approves it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "`<network_id>/<device_id>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"network_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.RegexMatches(networkIDPattern, "must be a 16 character lowercase hex network ID")},
			},
			"device_id": schema.StringAttribute{
				Required:      true,
				Description:   "ZeroTier address of the device (10 lowercase hex characters).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.RegexMatches(deviceIDPattern, "must be a 10 character lowercase hex ZeroTier address")},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Name of the member.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the member.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(centralv2.Authorized)),
				Description: "One of `authorized`, `not_authorized` or `rejected`. Defaults to `authorized`.",
				Validators: []validator.String{stringvalidator.OneOf(
					string(centralv2.Authorized), string(centralv2.NotAuthorized), string(centralv2.Rejected),
				)},
			},
			"active_bridge": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow the member to bridge other traffic onto the network.",
			},
			"no_auto_assign_ips": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Exempt the member from automatic IP assignment.",
			},
			"ipv4_assignments": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "IPv4 addresses assigned to the member. Left unset, Central auto-assigns from the network's pools. " +
					"Set `no_auto_assign_ips` when assigning an empty list.",
			},
			"ipv6_assignments": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "IPv6 addresses assigned to the member. Left unset, Central auto-assigns from the network's pools.",
			},
			"identity": schema.StringAttribute{
				Computed:    true,
				Description: "Public identity of the device, once it has joined.",
			},
			"ip_assignment_6plane": schema.StringAttribute{
				Computed:    true,
				Description: "6PLANE address, when the network assigns them.",
			},
			"ip_assignment_rfc4193": schema.StringAttribute{
				Computed:    true,
				Description: "RFC 4193 address, when the network assigns them.",
			},
			"creation_time": schema.StringAttribute{
				Computed:      true,
				Description:   "When the member was created (RFC 3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan memberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	member := r.apply(ctx, api, plan, &resp.Diagnostics)
	if member == nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, memberToModel(member))...)
}

func (r *memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state memberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.GetNetworkMemberWithResponse(ctx, state.NetworkID.ValueString(), state.DeviceID.ValueString())
	member, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		if removeIfGone(ctx, err, "Member", state.ID.ValueString(), resp) {
			return
		}
		resp.Diagnostics.AddError("Unable to read member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, memberToModel(member))...)
}

func (r *memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan memberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	member := r.apply(ctx, api, plan, &resp.Diagnostics)
	if member == nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, memberToModel(member))...)
}

func (r *memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state memberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.RemoveNetworkMembersWithResponse(ctx, state.NetworkID.ValueString(), []centralv2.DeleteMembersRequest{
		{DeviceId: state.DeviceID.ValueString()},
	})
	if err := checkNoBody(err, deref(res).HTTPResponse, deref(res).Body); err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Unable to delete member", err.Error())
	}
}

// ImportState accepts `<network_id>/<device_id>` or `<network_id>-<device_id>`.
func (r *memberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	networkID, deviceID, ok := strings.Cut(req.ID, "/")
	if !ok {
		networkID, deviceID, ok = strings.Cut(req.ID, "-")
	}
	if !ok || !networkIDPattern.MatchString(networkID) || !deviceIDPattern.MatchString(deviceID) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected <network_id>/<device_id>, got %q", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_id"), deviceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), networkID+"/"+deviceID)...)
}

// apply writes the member's settings (the update endpoint also creates it),
// then moves it to the planned status, which has its own endpoints.
func (r *memberResource) apply(ctx context.Context, api *centralv2.ClientWithResponses, plan memberModel, diags *diag.Diagnostics) *centralv2.NetworkMember {
	networkID, deviceID := plan.NetworkID.ValueString(), plan.DeviceID.ValueString()

	body := centralv2.UpdateMemberRequest{
		Name:            ptr(plan.Name.ValueString()),
		Description:     ptr(plan.Description.ValueString()),
		ActiveBridge:    ptr(plan.ActiveBridge.ValueBool()),
		NoAutoAssignIps: ptr(plan.NoAutoAssignIPs.ValueBool()),
	}
	if known(plan.IPv4Assignments) {
		body.Ipv4Assignments = ptr(stringsFromList(ctx, plan.IPv4Assignments, diags))
	}
	if known(plan.IPv6Assignments) {
		body.Ipv6Assignments = ptr(stringsFromList(ctx, plan.IPv6Assignments, diags))
	}
	if diags.HasError() {
		return nil
	}

	res, err := api.UpdateNetworkMemberWithResponse(ctx, networkID, deviceID, body)
	member, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		diags.AddError("Unable to update member", err.Error())
		return nil
	}

	want := centralv2.NetworkMemberStatus(plan.Status.ValueString())
	if member.Status == want {
		return member
	}

	member, err = setMemberStatus(ctx, api, networkID, deviceID, want)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to set member status to %s", want), err.Error())
		return nil
	}

	return member
}

func setMemberStatus(ctx context.Context, api *centralv2.ClientWithResponses, networkID, deviceID string, want centralv2.NetworkMemberStatus) (*centralv2.NetworkMember, error) {
	switch want {
	case centralv2.Authorized:
		res, err := api.AuthorizeNetworkMemberWithResponse(ctx, networkID, deviceID)
		return check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	case centralv2.NotAuthorized:
		res, err := api.DeauthorizeNetworkMemberWithResponse(ctx, networkID, deviceID)
		return check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	case centralv2.Rejected:
		res, err := api.RejectNetworkMemberWithResponse(ctx, networkID, deviceID)
		return check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	}
	return nil, fmt.Errorf("unknown member status %q", want)
}

func memberToModel(m *centralv2.NetworkMember) memberModel {
	return memberModel{
		ID:                  types.StringValue(m.NetworkId + "/" + m.DeviceId),
		NetworkID:           types.StringValue(m.NetworkId),
		DeviceID:            types.StringValue(m.DeviceId),
		Name:                types.StringValue(m.Name),
		Description:         types.StringValue(m.Description),
		Status:              types.StringValue(string(m.Status)),
		ActiveBridge:        types.BoolValue(m.ActiveBridge),
		NoAutoAssignIPs:     types.BoolValue(m.NoAutoAssignIps),
		IPv4Assignments:     stringsToList(m.Ipv4Assignments),
		IPv6Assignments:     stringsToList(m.Ipv6Assignments),
		Identity:            types.StringValue(m.Identity),
		IPAssignment6plane:  types.StringValue(m.IpAssignment6plane),
		IPAssignmentRFC4193: types.StringValue(m.IpAssignmentRfc4193),
		CreationTime:        timeString(m.CreationTime),
	}
}

func stringsFromList(ctx context.Context, l types.List, diags *diag.Diagnostics) []string {
	out := []string{}
	diags.Append(l.ElementsAs(ctx, &out, false)...)
	return out
}

func stringsToList(s []string) types.List {
	elems := make([]attr.Value, 0, len(s))
	for _, v := range s {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}
