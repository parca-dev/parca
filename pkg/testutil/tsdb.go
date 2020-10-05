package testutil

import (
	"github.com/conprof/db/tsdb"
	"io/ioutil"
	"math"
)

func NewTSDB() (*tsdb.DB, error) {
	dir, err := ioutil.TempDir("", "conprof-test")
	if err != nil {
		return nil, err
	}
	opts := tsdb.DefaultOptions()
	opts.RetentionDuration = math.MaxInt64
	return tsdb.Open(dir, nil, nil, opts)
}
