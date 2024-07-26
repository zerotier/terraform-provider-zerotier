package zerotier

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		Schema:        buildMemberSchema(true),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

//
// CRUD
//

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	nwid, nodeId, err := resourceNetworkAndNodeIdentifiers(d)
	if err != nil {
		return diag.FromErr(err)
	}

	member, err := c.GetMember(ctx, nwid, nodeId)
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

func resourceNetworkAndNodeIdentifiers(d *schema.ResourceData) (string, string, error) {
	nwid := d.Get("network_id").(string)
	nodeID := d.Get("member_id").(string)

	if nwid == "" && nodeID == "" {
		var err error
		nwid, nodeID, err = parseMemberId(d.Id())
		if err != nil {
			return "", "", err
		}
	}
	return nwid, nodeID, nil
}

func parseMemberId(id string) (string, string, error) {
	parts := strings.SplitN(id, "-", 2)

	if len(parts) != 2 {
		return "", "", errors.New(fmt.Sprintf("invalid format: '%s' (wrong syntax)", id))
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", errors.New(fmt.Sprintf("invalid format: '%s' (all components are required)", id))
	}

	return parts[0], parts[1], nil
}
