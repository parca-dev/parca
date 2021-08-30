package queryv1alpha1

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
		validation.Field(&r.Options, validation.Required, optionMatchesProfileMode(r.Mode)),
	)
	if err != nil {
		return err
	}

	switch r.Mode {
	case QueryRequest_MODE_SINGLE_UNSPECIFIED:
		err := validateSingle(r.GetSingle())
		if err != nil {
			return err
		}
	case QueryRequest_MODE_DIFF:
		err := validateDiff(r.GetDiff())
		if err != nil {
			return err
		}
	case QueryRequest_MODE_MERGE:
		err := validateMerge(r.GetMerge())
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid mode")
	}

	return nil
}

func validateSingle(single *SingleProfile) error {
	if single == nil {
		return fmt.Errorf("single must not be unset")
	}

	return validation.ValidateStruct(single,
		validation.Field(&single.Time, validation.Required),
		validation.Field(&single.Query, validation.Required),
	)
}

func validateMerge(merge *MergeProfile) error {
	if merge == nil {
		return fmt.Errorf("merge must not be unset")
	}

	return validation.ValidateStruct(merge,
		validation.Field(&merge.Start, validation.Required),
		validation.Field(&merge.End, validation.Required, isAfter(merge.Start)),
		validation.Field(&merge.Query, validation.Required),
	)
}

func validateDiff(diff *DiffProfile) error {
	if diff == nil {
		return fmt.Errorf("diff must not be unset")
	}

	err := validation.ValidateStruct(diff,
		validation.Field(&diff.A, validation.Required),
		validation.Field(&diff.B, validation.Required),
	)
	if err != nil {
		return err
	}

	err = validateProfileSelection(diff.A)
	if err != nil {
		return err
	}

	err = validateProfileSelection(diff.B)
	if err != nil {
		return err
	}

	return nil
}

func validateProfileSelection(sel *ProfileDiffSelection) error {
	err := validation.ValidateStruct(sel,
		validation.Field(&sel.Options, validation.Required, optionMatchesDiffProfileSelectionMode(sel.Mode)),
	)
	if err != nil {
		return err
	}

	switch sel.Mode {
	case ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		err := validateSingle(sel.GetSingle())
		if err != nil {
			return err
		}
	case ProfileDiffSelection_MODE_MERGE:
		err := validateMerge(sel.GetMerge())
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid mode")
	}

	return nil
}

func optionMatchesDiffProfileSelectionMode(mode ProfileDiffSelection_Mode) DiffProfileSelectionOptionMatchesRule {
	return DiffProfileSelectionOptionMatchesRule{
		mode: mode,
	}
}

// DiffProfileSelectionOptionMatchesRule ensure the options match the requested mode
type DiffProfileSelectionOptionMatchesRule struct {
	mode ProfileDiffSelection_Mode
}

// Validate the option matches mode
func (o DiffProfileSelectionOptionMatchesRule) Validate(v interface{}) error {
	option, ok := v.(isProfileDiffSelection_Options)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	switch o.mode {
	case ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		if _, ok := option.(*ProfileDiffSelection_Single); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	case ProfileDiffSelection_MODE_MERGE:
		if _, ok := option.(*ProfileDiffSelection_Merge); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	default:
		return fmt.Errorf("invalid value")
	}
}

func optionMatchesProfileMode(mode QueryRequest_Mode) ProfileOptionMatchesRule {
	return ProfileOptionMatchesRule{
		mode: mode,
	}
}

// ProfileOptionMatchesRule ensure the options match the requested mode
type ProfileOptionMatchesRule struct {
	mode QueryRequest_Mode
}

// Validate the option matches mode
func (o ProfileOptionMatchesRule) Validate(v interface{}) error {
	option, ok := v.(isQueryRequest_Options)
	if !ok {
		return fmt.Errorf("invalid value")
	}

	switch o.mode {
	case QueryRequest_MODE_SINGLE_UNSPECIFIED:
		if _, ok := option.(*QueryRequest_Single); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	case QueryRequest_MODE_DIFF:
		if _, ok := option.(*QueryRequest_Diff); !ok {
			return fmt.Errorf("invalid option for mode")
		}
		return nil
	case QueryRequest_MODE_MERGE:
		if _, ok := option.(*QueryRequest_Merge); !ok {
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
