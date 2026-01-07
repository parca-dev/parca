// Copyright 2022-2026 The Parca Authors
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
	"fmt"
	"strings"

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

// ScrapeConfigsValid is the ValidRule.
var ScrapeConfigsValid = ScrapeConfigsValidRule{}

// ScrapeConfigsValidRule is a validation rule for the Config. It implements the validation.Rule interface.
type ScrapeConfigsValidRule struct{}

// ScrapeConfigsValidate returns an error if the config is not valid.
func (v ScrapeConfigsValidRule) Validate(value interface{}) error {
	sc, ok := value.([]*ScrapeConfig)
	if !ok {
		return errors.New("ScrapeConfigs array is invalid")
	}

	uniqueJobNames := map[string]int{}
	for _, c := range sc {
		if c != nil {
			uniqueJobNames[c.JobName]++
		}
	}

	duplicateJobNames := make([]string, 0)
	for jobName, count := range uniqueJobNames {
		if count > 1 {
			duplicateJobNames = append(duplicateJobNames, jobName)
		}
	}

	if len(duplicateJobNames) > 0 {
		return fmt.Errorf("duplicate job_name found in scrape configs: %s", strings.Join(duplicateJobNames, ", "))
	}

	return nil
}
