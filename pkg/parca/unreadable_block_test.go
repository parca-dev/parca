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

package parca

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-kit/log"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/providers/filesystem"
)

// scanBlock writes data.parquet holding data, or its first truncate bytes if truncate > 0, and scans it.
func scanBlock(t *testing.T, truncate int, mode os.FileMode) error {
	t.Helper()
	var buf bytes.Buffer
	w := parquet.NewGenericWriter[struct{ V int64 }](&buf, parquet.KeyValueMetadata(dynparquet.DynamicColumnsKey, ""))
	_, err := w.Write([]struct{ V int64 }{{V: 1}})
	require.NoError(t, err)
	require.NoError(t, w.Close())
	data := buf.Bytes()
	if truncate > 0 {
		data = data[:truncate]
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "db", "table", "01HZZZZZZZZZZZZZZZZZZZZZZ0", "data.parquet")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), os.ModePerm))
	require.NoError(t, os.WriteFile(file, data, mode))

	bkt, err := filesystem.NewBucket(dir)
	require.NoError(t, err)
	store := frostdb.NewDefaultObjstoreBucket(bkt, frostdb.StorageWithSkipUnreadableBlocks(skipUnreadableBlock(log.NewNopLogger())))
	return store.Scan(context.Background(), "db/table", nil, nil, 0, func(context.Context, any) error { return nil })
}

func TestSkipUnreadableBlock(t *testing.T) {
	require.NoError(t, scanBlock(t, 10, 0o644))

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		require.ErrorIs(t, scanBlock(t, 0, 0), os.ErrPermission)
	}
}
