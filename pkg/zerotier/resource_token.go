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
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "The name of the token; if you do not supply this value, one will be generated",
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

	name := d.Get("name").(string)

	if name == "" {
		var err error
		name, err = c.RandomToken(ctx)
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("name", name)
	}

	token, err := c.RandomToken(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.CreateAPIToken(ctx, *user.Id, name, token); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)
	d.Set("token", token)

	return nil
}

func resourceTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	user, err := c.User(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteAPIToken(ctx, *user.Id, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
