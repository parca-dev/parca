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

package ui

import "embed"

//go:embed packages/app/web/dist
//go:embed packages/app/web/dist/_next
//go:embed packages/app/web/dist/_next/static/chunks/pages/*.js
//go:embed packages/app/web/dist/_next/static/*/*.js
var FS embed.FS

// NOTICE: Static HTML export of a Next.js app contains several files prefixed with _,
// directives for all these patterns need to explicitly added.
// > If a pattern names a directory, all files in the subtree rooted at that directory are embedded (recursively),
// > except that files with names beginning with ‘.’ or ‘_’ are excluded.
// source: https://pkg.go.dev/embed
