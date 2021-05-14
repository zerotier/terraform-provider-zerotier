package zerotier

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func strNonEmpty(i interface{}) diag.Diagnostics {
	switch i := i.(type) {
	case *string:
		if i == nil || strings.TrimSpace(*i) == "" {
			return diag.FromErr(errors.New("value is an empty string"))
		}
	case string:
		if strings.TrimSpace(i) == "" {
			return diag.FromErr(errors.New("value is an empty string"))
		}
	default:
		return diag.FromErr(errors.New("not a string"))
	}

	return nil
}
