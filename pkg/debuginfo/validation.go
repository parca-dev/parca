package debuginfo

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/thanos-io/thanos/pkg/objstore/client"
)

// Valid is the ValidRule
var Valid = ValidRule{}

// ValidRule is a validation rule for the Config. It implementes the validation.Rule interface
type ValidRule struct{}

// Validate returns an error if the config is not valid
func (v ValidRule) Validate(value interface{}) error {
	c, ok := value.(*Config)
	if !ok {
		return errors.New("DebugInfo is invalid")
	}
	return validation.ValidateStruct(c,
		validation.Field(c.Bucket, validation.Required, BucketValid),
	)
}

var BucketValid = BucketRule{}

type BucketRule struct{}

// Validate the bucket config
func (r BucketRule) Validate(value interface{}) error {
	b, ok := value.(*client.BucketConfig)
	if !ok {
		return errors.New("BucketConfig is invalid")
	}

	return validation.ValidateStruct(b,
		validation.Field(&b.Type, validation.Required),
		validation.Field(&b.Config, validation.Required),
	)
}
