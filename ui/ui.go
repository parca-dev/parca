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
