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

package query

import (
	"testing"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
)

func BenchmarkStringMatching(b *testing.B) {
	testValue := []byte("runtime.goexit")
	testTarget := []byte("goexit")

	b.Run("equalFoldBytes", func(b *testing.B) {
		condition := &pb.StringCondition{
			Condition: &pb.StringCondition_Equal{
				Equal: string(testTarget),
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matchesStringCondition(testValue, condition)
		}
	})

	b.Run("containsFoldBytes", func(b *testing.B) {
		condition := &pb.StringCondition{
			Condition: &pb.StringCondition_Contains{
				Contains: string(testTarget),
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matchesStringCondition(testValue, condition)
		}
	})

	b.Run("hasPrefixFoldBytes", func(b *testing.B) {
		condition := &pb.StringCondition{
			Condition: &pb.StringCondition_StartsWith{
				StartsWith: "runtime",
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matchesStringCondition(testValue, condition)
		}
	})
}

func BenchmarkStringMatchingLongStrings(b *testing.B) {
	testValue := []byte("github.com/parca-dev/parca/pkg/profilestore/profilestoreserver.(*ProfileStoreServer).WriteRaw")
	testTarget := []byte("profilestore")

	b.Run("containsFoldBytes", func(b *testing.B) {
		condition := &pb.StringCondition{
			Condition: &pb.StringCondition_Contains{
				Contains: string(testTarget),
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matchesStringCondition(testValue, condition)
		}
	})
}
