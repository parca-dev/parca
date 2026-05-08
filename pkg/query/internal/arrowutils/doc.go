// Package arrowutils vendors the Arrow record sort/take/merge utilities
// from github.com/polarsignals/frostdb/pqarrow/arrowutils. These power
// the sort and merge step in the columnar query path.
//
// The merge_test.go and schema_test.go suites from upstream have been
// omitted because they depend on github.com/polarsignals/frostdb/internal/records,
// which is not importable from outside the frostdb module. sort_test.go
// is preserved.
package arrowutils
