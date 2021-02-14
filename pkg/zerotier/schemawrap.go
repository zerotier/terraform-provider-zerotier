package zerotier

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ValidatedSchema is an internal schema for validating and managing lots of
// schema parameters. It is intended to be a more-or-less write-through cache
// of terraform information with validation and conversion along the way.
type ValidatedSchema struct {
	// Schema is our schema. The key is the name. See SchemaWrap for more information.
	Schema map[string]*SchemaWrap
	// Should be programmed to yield the type at Yield time.
	YieldFunc func(ValidatedSchema) interface{}
	// Should be programmed to populate the validated schema with Set calls.
	CollectFunc func(ValidatedSchema, *schema.ResourceData, interface{}) diag.Diagnostics
}

// SchemaWrap wraps the terraform schema with validators and converters.
type SchemaWrap struct {
	// Schema is the terraform schema.
	Schema *schema.Schema
	// ValidatorFunc is a function, that if supplied, validates the data and
	// yields an error if the function returns one.
	ValidatorFunc func(interface{}) diag.Diagnostics
	// FromTerraformFunc converts data from terraform plans to the Value (see
	// below). It returns an error if it had trouble.
	FromTerraformFunc func(interface{}) (interface{}, diag.Diagnostics)
	// ToTerraformFunc converts data from the Value to the terraform
	// representation. This must *always* succeed (in practice, this has not been
	// an issue at this time)
	ToTerraformFunc func(interface{}) interface{}
	// EqualFunc is used in comparisons, which are used in determining if changes
	// need to be pushed to our API.
	EqualFunc func(interface{}, interface{}) bool
	// Value is the internal value; this is a representation suitable for using
	// in both ValidatedSchema.YieldFunc() and ValidatedSchema.CollectFunc
	// interchangeably, as in, they can be type asserted without panicking.
	Value interface{}
}

func (sw *SchemaWrap) Clone() *SchemaWrap {
	val, err := sw.Schema.DefaultValue()
	if err != nil {
		panic(err)
	}

	return &SchemaWrap{
		Value:             val,
		Schema:            sw.Schema,
		ValidatorFunc:     sw.ValidatorFunc,
		FromTerraformFunc: sw.FromTerraformFunc,
		ToTerraformFunc:   sw.ToTerraformFunc,
		EqualFunc:         sw.EqualFunc,
	}
}

func (vs ValidatedSchema) Clone() ValidatedSchema {
	vs2 := ValidatedSchema{
		Schema:      map[string]*SchemaWrap{},
		YieldFunc:   vs.YieldFunc,
		CollectFunc: vs.CollectFunc,
	}

	for key, sw := range vs.Schema {
		vs2.Schema[key] = sw.Clone()
	}

	return vs2
}

// TerraformSchema returns the unadulterated schema for use by terraform.
func (vs ValidatedSchema) TerraformSchema() map[string]*schema.Schema {
	res := map[string]*schema.Schema{}

	for k, v := range vs.Schema {
		res[k] = v.Schema
	}

	return res
}

// CollectFromTerraform collects all the properties listed in the validated schema, converts
// & validates them, and makes this object available for further use. Failure
// to call this method before others on the same transaction may result in
// undefined behavior.
func (vs ValidatedSchema) CollectFromTerraform(d *schema.ResourceData) diag.Diagnostics {
	for key, sw := range vs.Schema {
		var (
			err diag.Diagnostics
		)

		res, ok := d.GetOk(key)
		if !ok {
			if sw.Value == nil {
				res = d.Get(key)
			} else {
				continue
			}
		}

		if sw.FromTerraformFunc != nil {
			if res, err = sw.FromTerraformFunc(res); err != nil {
				return err
			}
		}

		if sw.ValidatorFunc != nil {
			if err := sw.ValidatorFunc(res); err != nil {
				return err
			}
		}

		sw.Value = res
	}

	return nil
}

// CollectFromObject is a pre-programmed call on the struct which accepts the
// known object and sets all the values appropriately.
func (vs ValidatedSchema) CollectFromObject(d *schema.ResourceData, i interface{}) diag.Diagnostics {
	return vs.CollectFunc(vs, d, i)
}

// Get retrieves the set value inside the schema.
func (vs ValidatedSchema) Get(key string) interface{} {
	return vs.Schema[key].Value
}

// Set a value in terraform. This goes through our validation & conversion
// first.
func (vs ValidatedSchema) Set(d *schema.ResourceData, key string, value interface{}) diag.Diagnostics {
	sw := vs.Schema[key]
	if sw == nil {
		return diag.FromErr(errors.New("invalid key, plugin error"))
	}

	if sw.ValidatorFunc != nil {
		if err := sw.ValidatorFunc(value); err != nil {
			return err
		}
	}

	if sw.ToTerraformFunc != nil {
		value = sw.ToTerraformFunc(value)
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}

	sw.Value = value

	return nil
}

// RemoteChanged reports if our data source has changed.
// FIXME probably doesn't work yet; coming soon! :D
func (vs ValidatedSchema) RemoteChanged(d *schema.ResourceData) bool {
	for key, sw := range vs.Schema {
		if !sw.EqualFunc(sw.Value, d.Get(key)) {
			return true
		}
	}

	return false
}

// Yield yields the type on request.
func (vs ValidatedSchema) Yield() interface{} {
	return vs.YieldFunc(vs)
}
