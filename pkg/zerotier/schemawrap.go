package zerotier

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ValidatedSchema is an internal schema for validating and managing lots of
// schema parameters.
type ValidatedSchema struct {
	Schema map[string]*SchemaWrap
	// Should be programmed to yield the type at Yield time.
	YieldFunc func(ValidatedSchema) interface{}
	// Should be programmed to populate the validated schema with Set calls.
	CollectFunc func(ValidatedSchema, *schema.ResourceData, interface{}) diag.Diagnostics
}

// SchemaWrap wraps the terraform schema with validators and converters.
type SchemaWrap struct {
	Schema            *schema.Schema
	ValidatorFunc     func(interface{}) diag.Diagnostics
	FromTerraformFunc func(interface{}) (interface{}, diag.Diagnostics)
	ToTerraformFunc   func(interface{}) interface{}
	EqualFunc         func(interface{}, interface{}) bool
	Value             interface{}
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
			res interface{}
			err diag.Diagnostics
		)

		if sw.FromTerraformFunc != nil {
			if res, err = sw.FromTerraformFunc(d.Get(key)); err != nil {
				return err
			}
		} else {
			res = d.Get(key)
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
