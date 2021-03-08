package zerotier

import (

	//	"strconv"
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
		Schema:        ZTMember.TerraformSchema(),
	}
}

//
// CRUD
//

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := ZTMember.Clone()
	c := m.(*ztcentral.Client)

	ztNetworkID, ztNodeID := getMemberIDs(d)
	member, err := c.GetMember(ctx, ztNetworkID, ztNodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	return ztm.CollectFromObject(d, member)
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := ZTMember.Clone()
	if err := ztm.CollectFromTerraform(d); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	member := ztm.Yield().(*ztcentral.Member)

	cm, err := c.CreateAuthorizedMember(ctx, member.NetworkID, member.MemberID, member.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cm.ID)
	return nil
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := ZTMember.Clone()

	if err := resourceMemberRead(ctx, d, m); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	ztm.CollectFromTerraform(d)

	updated, err := c.UpdateMember(ctx, ztm.Yield().(*ztcentral.Member))
	if err != nil {
		return diag.FromErr(err)
	}

	ztm.CollectFromObject(d, updated)

	return nil
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztm := ZTMember.Clone()
	ztm.CollectFromTerraform(d)

	c := m.(*ztcentral.Client)

	if err := c.DeleteMember(ctx, ztm.Yield().(*ztcentral.Member)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
