// Copyright 2021 The Parca Authors
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

package demangle

import (
	"testing"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"

	"github.com/stretchr/testify/require"
)

// Basic demangler tests to make sure that demangling is not completely
// broken for C++ or Rust.

func TestDemanglerAlreadyDemangled(t *testing.T) {
	demangler := NewDemangler("simple", true)

	function := pb.Function{
		SystemName: "main",
	}
	expected_function := pb.Function{
		Name:       "main",
		SystemName: "main",
	}

	demangled := demangler.Demangle(&function)
	require.Equal(t, &expected_function, demangled)
}

func TestDemanglerSimpleCppDemangling(t *testing.T) {
	demangler := NewDemangler("simple", true)

	function := pb.Function{
		SystemName: "_ZNSaIcEC1ERKS_",
	}
	expected_function := pb.Function{
		Name:       "std::allocator::allocator",
		SystemName: "_ZNSaIcEC1ERKS_",
	}

	demangled := demangler.Demangle(&function)
	require.Equal(t, &expected_function, demangled)
}

func TestDemangleNone(t *testing.T) {
	demangler := NewDemangler("none", true)

	function := pb.Function{
		SystemName: "_ZNSaIcEC1ERKS_",
	}
	expected_function := pb.Function{
		SystemName: "_ZNSaIcEC1ERKS_",
	}

	demangled := demangler.Demangle(&function)
	require.Equal(t, &expected_function, demangled)
}

func TestDemanglerTemplatesCppDemangling(t *testing.T) {
	demangler := NewDemangler("templates", true)

	function := pb.Function{
		SystemName: "_ZNSaIcEC1ERKS_",
	}
	expected_function := pb.Function{
		Name:       "std::allocator<char>::allocator",
		SystemName: "_ZNSaIcEC1ERKS_",
	}

	demangled := demangler.Demangle(&function)
	require.Equal(t, &expected_function, demangled)
}

func TestDemanglerSimpleRustDemangling(t *testing.T) {
	demangler := NewDemangler("simple", true)

	function := pb.Function{
		SystemName: "_ZN11collections5slice29_$LT$impl$u20$$u5b$T$u5d$$GT$10as_mut_ptr17hf12a6d0409938c96E",
	}
	expected_function := pb.Function{
		Name:       "collections::slice::<impl [T]>::as_mut_ptr",
		SystemName: "_ZN11collections5slice29_$LT$impl$u20$$u5b$T$u5d$$GT$10as_mut_ptr17hf12a6d0409938c96E",
	}

	demangled := demangler.Demangle(&function)
	require.Equal(t, &expected_function, demangled)
}
