package zerotier

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func strNonEmpty(i interface{}) diag.Diagnostics {
	if strings.TrimSpace(i.(string)) == "" {
		return diag.FromErr(errors.New("value is an empty string"))
	}

	return nil
}
