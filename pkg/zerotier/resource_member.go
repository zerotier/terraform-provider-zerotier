package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceMember() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage ZeroTier members and join them to networks",
		CreateContext: resourceMemberCreate,
		ReadContext:   resourceMemberRead,
		UpdateContext: resourceMemberUpdate,
		DeleteContext: resourceMemberDelete,
		Schema:        MemberSchema,
	}
}

//
// CRUD
//

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	member, err := c.GetMember(ctx, d.Get("network_id").(string), d.Get("member_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	return memberToTerraform(d, member)
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	member := toMember(d)
	c := m.(*ztcentral.Client)

	_, err := c.CreateAuthorizedMember(ctx, *member.NetworkId, *member.NodeId, *member.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := c.UpdateMember(ctx, *member.NetworkId, *member.NodeId, member)
	if err != nil {
		return diag.FromErr(err)
	}

	return memberToTerraform(d, res)
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	member := toMember(d)

	updated, err := c.UpdateMember(ctx, *member.NetworkId, *member.NodeId, member)
	if err != nil {
		return diag.FromErr(err)
	}

	return memberToTerraform(d, updated)
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	member := toMember(d)

	if err := c.DeleteMember(ctx, *member.NetworkId, *member.NodeId); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
