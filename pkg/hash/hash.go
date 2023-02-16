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
//

package hash

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// File returns the hash of the file.
func File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open the file: %w", err)
	}
	defer f.Close()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash the file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Reader returns the hash of the reader.
func Reader(r io.Reader) (string, error) {
	h := xxhash.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash the reader: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
