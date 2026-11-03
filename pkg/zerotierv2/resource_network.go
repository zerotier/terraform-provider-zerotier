package zerotierv2

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zerotier/terraform-provider-zerotier/pkg/centralv2"
)

var (
	_ resource.ResourceWithConfigure      = &networkResource{}
	_ resource.ResourceWithImportState    = &networkResource{}
	_ resource.ResourceWithValidateConfig = &networkResource{}
)

// Central V2 defaults (core/network.NewNetworkConfig in central-v2).
const (
	defaultMTU            = 2800
	defaultMulticastLimit = 32
)

// The provider is served on plugin protocol 5 so it keeps working with every
// Terraform version the V1 resources support. Protocol 5 has no nested
// attributes, so lists of ranges and routes are list(object) attributes and
// the DNS and assignment-mode objects are flattened.
var (
	ipRangeType = types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.StringType, "end": types.StringType}}
	routeType   = types.ObjectType{AttrTypes: map[string]attr.Type{"target": types.StringType, "via": types.StringType}}
)

type networkResource struct {
	baseResource
}

type networkModel struct {
	ID                types.String `tfsdk:"id"`
	NetworkGroupID    types.String `tfsdk:"network_group_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Kind              types.String `tfsdk:"kind"`
	CreationTime      types.String `tfsdk:"creation_time"`
	MTU               types.Int64  `tfsdk:"mtu"`
	MulticastLimit    types.Int64  `tfsdk:"multicast_limit"`
	EnableBroadcast   types.Bool   `tfsdk:"enable_broadcast"`
	Private           types.Bool   `tfsdk:"private"`
	V4Subnet          types.String `tfsdk:"v4_subnet"`
	V6Subnet          types.String `tfsdk:"v6_subnet"`
	V4AssignmentPools types.List   `tfsdk:"v4_assignment_pools"`
	V6AssignmentPools types.List   `tfsdk:"v6_assignment_pools"`
	Routes            types.List   `tfsdk:"routes"`
	DNSDomain         types.String `tfsdk:"dns_domain"`
	DNSServers        types.List   `tfsdk:"dns_servers"`
	V4AutoAssign      types.Bool   `tfsdk:"v4_auto_assign"`
	V6AutoAssign      types.Bool   `tfsdk:"v6_auto_assign"`
	V6Assign6plane    types.Bool   `tfsdk:"v6_assign_6plane"`
	V6AssignRFC4193   types.Bool   `tfsdk:"v6_assign_rfc4193"`
}

type ipRangeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type routeModel struct {
	Target types.String `tfsdk:"target"`
	Via    types.String `tfsdk:"via"`
}

func newNetworkResource() resource.Resource {
	return &networkResource{}
}

func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_network"
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Central V2 network, created inside a network group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Network ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"network_group_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the network group the network belongs to. Changing it creates a new network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the network.",
				Validators:  []validator.String{stringvalidator.LengthBetween(3, 127)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the network.",
				Validators:  []validator.String{stringvalidator.LengthAtMost(1024)},
			},
			"kind": schema.StringAttribute{
				Computed:      true,
				Description:   "Kind of network. `v1` is a ZeroTier One network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"creation_time": schema.StringAttribute{
				Computed:      true,
				Description:   "When the network was created (RFC 3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"mtu": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultMTU),
				Description: "MTU of the virtual network adapter on members.",
				Validators:  []validator.Int64{int64validator.Between(1280, 10000)},
			},
			"multicast_limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultMulticastLimit),
				Description: "Maximum recipients of a multicast or broadcast packet. Must be 0 when `enable_broadcast` is false.",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"enable_broadcast": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Allow Ethernet broadcast (ff:ff:ff:ff:ff:ff).",
			},
			"private": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether members must be authorized to join. If false, any device can join.",
			},
			"v4_subnet": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "IPv4 subnet (CIDR) of the network. Central picks a random one when unset; set `v4_assignment_pools` with it.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"v6_subnet": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "IPv6 subnet (canonical CIDR) of the network.",
			},
			"v4_assignment_pools": schema.ListAttribute{
				ElementType:   ipRangeType,
				Optional:      true,
				Computed:      true,
				Description:   "IPv4 ranges (`{ start, end }`) addresses are auto-assigned from. Must fall inside `v4_subnet`. Central picks one when unset.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"v6_assignment_pools": schema.ListAttribute{
				ElementType: ipRangeType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(ipRangeType, nil)),
				Description: "IPv6 ranges (`{ start, end }`) addresses are auto-assigned from. Must fall inside `v6_subnet`.",
			},
			"routes": schema.ListAttribute{
				ElementType: routeType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(routeType, nil)),
				Description: "Additional routes (`{ target, via }`) pushed to members.",
			},
			"dns_domain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "DNS search domain pushed to members. Required when `dns_servers` is set.",
			},
			"dns_servers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, nil)),
				Description: "DNS server addresses pushed to members (up to 10).",
				Validators:  []validator.List{listvalidator.SizeAtMost(10)},
			},
			"v4_auto_assign": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Auto-assign IPv4 addresses from `v4_assignment_pools`.",
			},
			"v6_auto_assign": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Auto-assign IPv6 addresses from `v6_assignment_pools`.",
			},
			"v6_assign_6plane": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Assign each member a 6PLANE /80.",
			},
			"v6_assign_rfc4193": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Assign each member an RFC 4193 /128.",
			},
		},
	}
}

func (r *networkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg networkModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.EnableBroadcast.IsUnknown() || cfg.MulticastLimit.IsUnknown() {
		return
	}
	if !cfg.EnableBroadcast.IsNull() && !cfg.EnableBroadcast.ValueBool() &&
		(cfg.MulticastLimit.IsNull() || cfg.MulticastLimit.ValueInt64() != 0) {
		resp.Diagnostics.AddAttributeError(path.Root("multicast_limit"), "multicast_limit must be 0",
			"Central V2 requires multicast_limit = 0 when enable_broadcast is false.")
	}
}

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := networkConfigRequest(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// A create request takes either no config or every config field, and the
	// subnet and pool are optional here. So create with Central's defaults,
	// then apply the configured fields as an update.
	created, err := api.CreateNetworkWithResponse(ctx, plan.NetworkGroupID.ValueString(), centralv2.CreateNetworkRequest{
		Name:        plan.Name.ValueString(),
		Description: ptr(plan.Description.ValueString()),
	})
	network, err := check(err, deref(created).JSON200, deref(created).HTTPResponse, deref(created).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create network", err.Error())
		return
	}

	// Save the ID now so a failed update leaves a tainted resource rather than
	// an orphaned network.
	resp.Diagnostics.Append(resp.State.Set(ctx, networkToModel(network))...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := api.UpdateNetworkWithResponse(ctx, network.Id, centralv2.UpdateNetworkRequest{Config: cfg})
	network, err = check(err, deref(updated).JSON200, deref(updated).HTTPResponse, deref(updated).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure network", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkToModel(network))...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.GetNetworkWithResponse(ctx, state.ID.ValueString())
	network, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		if removeIfGone(ctx, err, "Network", state.ID.ValueString(), resp) {
			return
		}
		resp.Diagnostics.AddError("Unable to read network", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkToModel(network))...)
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state networkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := networkConfigRequest(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.UpdateNetworkWithResponse(ctx, state.ID.ValueString(), centralv2.UpdateNetworkRequest{
		Name:        ptr(plan.Name.ValueString()),
		Description: ptr(plan.Description.ValueString()),
		Config:      cfg,
	})
	network, err := check(err, deref(res).JSON200, deref(res).HTTPResponse, deref(res).Body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update network", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, networkToModel(network))...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := r.api(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := api.DeleteNetworkWithResponse(ctx, state.ID.ValueString())
	if err := checkNoBody(err, deref(res).HTTPResponse, deref(res).Body); err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Unable to delete network", err.Error())
	}
}

func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// networkConfigRequest builds the full config from the plan. Fields that are
// still unknown (the subnet and pool Central picks on create) are omitted so
// Central keeps its current values.
func networkConfigRequest(ctx context.Context, plan networkModel, diags *diag.Diagnostics) *centralv2.NetworkConfigRequest {
	cfg := &centralv2.NetworkConfigRequest{
		Mtu:             ptr(int(plan.MTU.ValueInt64())),
		MulticastLimit:  ptr(int(plan.MulticastLimit.ValueInt64())),
		EnableBroadcast: ptr(plan.EnableBroadcast.ValueBool()),
		Private:         ptr(plan.Private.ValueBool()),
		V6Subnet:        ptr(plan.V6Subnet.ValueString()),
		Dns: &centralv2.DNS{
			Domain:  plan.DNSDomain.ValueString(),
			Servers: stringsFromList(ctx, plan.DNSServers, diags),
		},
		V4AssignmentMode: &centralv2.IPV4AssignMode{Zt: plan.V4AutoAssign.ValueBool()},
		V6AssignmentMode: &centralv2.IPV6AssignMode{
			Zt:      plan.V6AutoAssign.ValueBool(),
			N6plane: plan.V6Assign6plane.ValueBool(),
			Rfc4193: plan.V6AssignRFC4193.ValueBool(),
		},
	}

	if known(plan.V4Subnet) {
		cfg.V4Subnet = ptr(plan.V4Subnet.ValueString())
	}
	if known(plan.V4AssignmentPools) {
		cfg.V4IpAssignmentPools = ptr(ipRangesFromList(ctx, plan.V4AssignmentPools, diags))
	}
	cfg.V6IpAssignmentPools = ptr(ipRangesFromList(ctx, plan.V6AssignmentPools, diags))

	var routes []routeModel
	diags.Append(plan.Routes.ElementsAs(ctx, &routes, false)...)
	apiRoutes := make([]centralv2.Route, 0, len(routes))
	for _, rt := range routes {
		apiRoutes = append(apiRoutes, centralv2.Route{Target: rt.Target.ValueString(), Via: rt.Via.ValueString()})
	}
	cfg.Routes = &apiRoutes

	return cfg
}

func ipRangesFromList(ctx context.Context, l types.List, diags *diag.Diagnostics) []centralv2.IPRange {
	var ranges []ipRangeModel
	diags.Append(l.ElementsAs(ctx, &ranges, false)...)
	out := make([]centralv2.IPRange, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, centralv2.IPRange{IpRangeStart: r.Start.ValueString(), IpRangeEnd: r.End.ValueString()})
	}
	return out
}

func networkToModel(n *centralv2.NetworkResponse) networkModel {
	c := n.Config

	routes := make([]attr.Value, 0, len(c.Routes))
	for _, rt := range c.Routes {
		routes = append(routes, types.ObjectValueMust(routeType.AttrTypes, map[string]attr.Value{
			"target": types.StringValue(rt.Target),
			"via":    types.StringValue(rt.Via),
		}))
	}

	return networkModel{
		ID:                types.StringValue(n.Id),
		NetworkGroupID:    types.StringValue(n.NetworkGroupId),
		Name:              types.StringValue(n.Name),
		Description:       types.StringValue(deref(n.Description)),
		Kind:              types.StringValue(n.Kind),
		CreationTime:      timeString(n.CreationTime),
		MTU:               types.Int64Value(int64(c.Mtu)),
		MulticastLimit:    types.Int64Value(int64(c.MulticastLimit)),
		EnableBroadcast:   types.BoolValue(c.EnableBroadcast),
		Private:           types.BoolValue(c.Private),
		V4Subnet:          types.StringValue(c.V4Subnet),
		V6Subnet:          types.StringValue(c.V6Subnet),
		V4AssignmentPools: ipRangesToList(c.V4IpAssignmentPools),
		V6AssignmentPools: ipRangesToList(c.V6IpAssignmentPools),
		Routes:            types.ListValueMust(routeType, routes),
		DNSDomain:         types.StringValue(c.Dns.Domain),
		DNSServers:        stringsToList(c.Dns.Servers),
		V4AutoAssign:      types.BoolValue(c.V4AssignmentMode.Zt),
		V6AutoAssign:      types.BoolValue(c.V6AssignmentMode.Zt),
		V6Assign6plane:    types.BoolValue(c.V6AssignmentMode.N6plane),
		V6AssignRFC4193:   types.BoolValue(c.V6AssignmentMode.Rfc4193),
	}
}

func ipRangesToList(ranges []centralv2.IPRange) types.List {
	elems := make([]attr.Value, 0, len(ranges))
	for _, r := range ranges {
		elems = append(elems, types.ObjectValueMust(ipRangeType.AttrTypes, map[string]attr.Value{
			"start": types.StringValue(r.IpRangeStart),
			"end":   types.StringValue(r.IpRangeEnd),
		}))
	}
	return types.ListValueMust(ipRangeType, elems)
}

func known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}
