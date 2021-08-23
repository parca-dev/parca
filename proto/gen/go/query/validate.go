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

// Validate the QueryRequest
func (r *QueryRequest) Validate() error {
	err := validation.ValidateStruct(r,
		validation.Field(&r.Options, validation.Required, optionMatchesMode(r.Mode)),
	)
	if err != nil {
		return err
	}

	switch r.Mode {
	case QueryRequest_SINGLE:
		//TODO
	case QueryRequest_DIFF:
		//TODO
	case QueryRequest_MERGE:
		merge := r.GetMerge()
		err = validation.ValidateStruct(merge,
			validation.Field(&merge.Start, validation.Required),
			validation.Field(&merge.End, validation.Required, isAfter(merge.Start)),
			validation.Field(&merge.Query, validation.Required),
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid mode")
	}

	return nil
}

func optionMatchesMode(mode QueryRequest_Mode) OptionMatchesRule {
	return OptionMatchesRule{
		mode: mode,
	}
}

// OptionMatchesRule ensure the options match the requested mode
type OptionMatchesRule struct {
	mode QueryRequest_Mode
}

// Validate the option matches mode
func (o OptionMatchesRule) Validate(v interface{}) error {
	option, ok := v.(isQueryRequest_Options)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	switch o.mode {
	case QueryRequest_SINGLE:
		if _, ok := option.(*QueryRequest_Single_); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	case QueryRequest_DIFF:
		if _, ok := option.(*QueryRequest_Diff_); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	case QueryRequest_MERGE:
		if _, ok := option.(*QueryRequest_Merge_); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	default:
		return fmt.Errorf("invalid value")
	}
}

func isEnum(enum map[int32]string) EnumRule {
	return EnumRule{
		enum: enum,
	}
}

// EnumRule checks that the provided value is in the enum map
type EnumRule struct {
	enum map[int32]string
}

// Validate the enum
func (e EnumRule) Validate(v interface{}) error {
	i, ok := v.(*int32)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	_, ok = e.enum[*i]
	if !ok {
		return fmt.Errorf("invalid value")
	}

	return nil
}

func isAfter(t *timestamppb.Timestamp) AfterRule {
	return AfterRule{
		Timestamp: t,
	}
}

// AfterRule validates that the timestamp is after the given value
type AfterRule struct {
	Timestamp *timestamppb.Timestamp
}

// Validate runs the validation function for the AfterRule
func (a AfterRule) Validate(t interface{}) error {
	end, ok := t.(*timestamppb.Timestamp)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	if a.Timestamp.AsTime().After(end.AsTime()) {
		return fmt.Errorf("start timestamp must be before end")
	}

	return nil
}
