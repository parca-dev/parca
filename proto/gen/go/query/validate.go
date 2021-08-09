package query

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Validate the QueryRangeRequest
func (r *QueryRangeRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Start, validation.Required),
		validation.Field(&r.End, validation.Required, isAfter(r.Start)),
		validation.Field(&r.Query, validation.Required),
	)
}

func isAfter(t *timestamppb.Timestamp) AfterRule {
	return AfterRule{
		After: t,
	}
}

// AfterRule validates that the timestamp is after the given value
type AfterRule struct {
	After *timestamppb.Timestamp
}

// Validate runs the validation function for the AfterRule
func (a AfterRule) Validate(t interface{}) error {
	start, ok := t.(*timestamppb.Timestamp)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	if a.After.AsTime().Before(start.AsTime()) {
		return fmt.Errorf("start timestamp must be before end")
	}

	return nil
}
