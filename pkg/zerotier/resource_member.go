package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func resourceMember() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage ZeroTier members and join them to networks",
		CreateContext: resourceMemberCreate,
		ReadContext:   resourceMemberRead,
		UpdateContext: resourceMemberUpdate,
		DeleteContext: resourceMemberDelete,
		Schema:        NewMember().TerraformSchema(),
	}
}

//
// CRUD
//

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := NewMember()
	c := m.(*ztcentral.Client)

	ztNetworkID, ztNodeID := getMemberIDs(d)
	member, err := c.GetMember(ctx, ztNetworkID, ztNodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	return ztm.CollectFromObject(d, member, true)
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := NewMember()
	if err := ztm.CollectFromTerraform(d); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	member := ztm.Yield().(*spec.Member)

	cm, err := c.CreateAuthorizedMember(ctx, *member.NetworkId, *member.NodeId, *member.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*cm.Id)
	return nil
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	ztm := NewMember()

	ztm.CollectFromTerraform(d)

	tfMember := ztm.Yield().(*spec.Member)
	updated, err := c.UpdateMember(ctx, *tfMember.NetworkId, *tfMember.NodeId, tfMember)
	if err != nil {
		return diag.FromErr(err)
	}

	return ztm.CollectFromObject(d, updated, true)
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := NewMember()
	ztm.CollectFromTerraform(d)

	c := m.(*ztcentral.Client)

	member := ztm.Yield().(*spec.Member)

	if err := c.DeleteMember(ctx, *member.NetworkId, *member.NodeId); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
