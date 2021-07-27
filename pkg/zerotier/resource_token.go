package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTokenCreate,
		ReadContext:   resourceTokenRead,
		DeleteContext: resourceTokenDelete,
		Description:   "Generate API tokens for Central.",
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "The name of the token",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The value of the token",
			},
		},
	}
}

func resourceTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	user, err := c.User(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("id").(string)

	token, err := c.RandomToken(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.CreateAPIToken(ctx, *user.Id, name, token); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(d.Get("id").(string))
	d.Set("token", token)

	return nil
}

func resourceTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if s, ok := d.Get("token").(string); ok && len(s) != 0 {
		return nil
	}

	return resourceTokenCreate(ctx, d, m)
}

func resourceTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	user, err := c.User(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("id").(string)

	if err := c.DeleteAPIToken(ctx, *user.Id, name); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
