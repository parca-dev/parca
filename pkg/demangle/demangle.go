// Copyright 2024-2026 The Parca Authors
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
	"fmt"
	"unsafe"

	"github.com/ianlancetaylor/demangle"
)

// Demangler demangles GCC/LLVM C++ and Rust symbol names.
//
// Demangling is the inverse process of mangling (encoding of each unique
// function and parameter list combination into a unique name for the linker).
// With mangling the linker can tell the difference between overloaded functions
// (they have the same name in the source code but different parameter lists).
type Demangler struct {
	options []demangle.Option
}

var (
	DemangleOptions = []string{
		"no_params",
		"no_template_params",
		"no_clones",
		"no_rust",
		"verbose",
		"llvm_style",
	}
	demangleOptionMappings = map[string]demangle.Option{
		DemangleOptions[0]: demangle.NoParams,
		DemangleOptions[1]: demangle.NoTemplateParams,
		DemangleOptions[2]: demangle.NoClones,
		DemangleOptions[3]: demangle.NoRust,
		DemangleOptions[4]: demangle.Verbose,
		DemangleOptions[5]: demangle.LLVMStyle,
	}
)

func stringsToDemanglerOptions(stringOptions []string) ([]demangle.Option, error) {
	res := []demangle.Option{}

	for _, str := range stringOptions {
		opt, ok := demangleOptionMappings[str]
		if !ok {
			return nil, fmt.Errorf("unknown demangle option %q", opt)
		}
		res = append(res, opt)
	}

	return res, nil
}

func MustNewDefaultDemangler() Demangler {
	d, err := NewDemangler()
	if err != nil {
		panic(err)
	}
	return d
}

func NewDefaultDemangler() (Demangler, error) {
	return NewDemangler("no_params", "no_template_params")
}

// NewDemangler creates a new Demangler with a given demangler options.
func NewDemangler(options ...string) (Demangler, error) {
	demanglerOptions, err := stringsToDemanglerOptions(options)
	if err != nil {
		return Demangler{}, err
	}

	return newDemangler(demanglerOptions), nil
}

// newDemangler creates a new Demangler with a given demangler options.
func newDemangler(options []demangle.Option) Demangler {
	return Demangler{
		options: options,
	}
}

// Demangle updates the function names in a profile demangling C++ and
// Rust names, simplified according to demanglerMode. If force is set,
// overwrite any names that appear already demangled.
// A modified version of pprof demangler.
func (d Demangler) Demangle(fn []byte) string {
	if demangled := demangle.Filter(unsafeString(fn), d.options...); demangled != unsafeString(fn) {
		return demangled
	}
	return string(fn)
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
