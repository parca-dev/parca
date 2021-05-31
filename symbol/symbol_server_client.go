// Copyright 2021 The conprof Authors
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

package symbol

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type SymbolServerClient struct {
	url string
}

func NewSymbolServerClient(url string) *SymbolServerClient {
	return &SymbolServerClient{
		url: url,
	}
}

type Module struct {
	Type      string `json:"type,omitempty"`
	CodeID    string `json:"code_id,omitempty"`
	ImageAddr string `json:"image_addr,omitempty"`
}

type Stacktrace struct {
	Frames []Frame `json:"frames,omitempty"`
}

type Frame struct {
	Status          string `json:"status,omitempty"`
	Lang            string `json:"lang,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	SymAddr         string `json:"sym_addr,omitempty"`
	Function        string `json:"function,omitempty"`
	Filename        string `json:"filename,omitempty"`
	AbsPath         string `json:"abs_path,omitempty"`
	LineNo          int    `json:"lineno,omitempty"`
	InstructionAddr string `json:"instruction_addr,omitempty"`
}

type SymbolicateRequest struct {
	Modules     []Module     `json:"modules,omitempty"`
	Stacktraces []Stacktrace `json:"stacktraces,omitempty"`
}

type SymbolicateResponse struct {
	Modules     []Module     `json:"modules,omitempty"`
	Stacktraces []Stacktrace `json:"stacktraces,omitempty"`
}

func (c *SymbolServerClient) Symbolicate(ctx context.Context, r *SymbolicateRequest) (*SymbolicateResponse, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res SymbolicateResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
