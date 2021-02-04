package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztidentity"
)

func resourceIdentity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIdentityCreate,
		ReadContext:   resourceIdentityRead,
		DeleteContext: resourceIdentityDelete,

		Schema: map[string]*schema.Schema{
			"public_key": {
				Type:        schema.TypeString,
				Description: "public key",
				Computed:    true,
				ForceNew:    true,
			},
			"private_key": {
				Type:        schema.TypeString,
				Description: "private key",
				Computed:    true,
				ForceNew:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceIdentityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ident := ztidentity.NewZeroTierIdentity()

	d.SetId(ident.IDString())

	d.Set("public_key", ident.PublicKeyString())
	d.Set("private_key", ident.PrivateKeyString())

	return nil
}

func resourceIdentityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceIdentityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
