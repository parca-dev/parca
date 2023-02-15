// Copyright 2022-2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/thanos-io/objstore/client"
)

// ObjectStorageValid is the ValidRule.
var ObjectStorageValid = ObjectStorageValidRule{}

// ObjectStorageValidRule is a validation rule for the Config. It implements the validation.Rule interface.
type ObjectStorageValidRule struct{}

// ObjectStorageValidate returns an error if the config is not valid.
func (v ObjectStorageValidRule) Validate(value interface{}) error {
	c, ok := value.(*ObjectStorage)
	if !ok {
		return errors.New("debuginfod is invalid")
	}
	return validation.ValidateStruct(c,
		validation.Field(&c.Bucket, validation.Required, BucketValid),
	)
}

var BucketValid = BucketRule{}

type BucketRule struct{}

// Validate the bucket config.
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
