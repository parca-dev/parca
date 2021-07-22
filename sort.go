package storage

import (
	"sort"

	"github.com/google/pprof/profile"
)

func sortSamples(samples []*profile.Sample) {
	sort.Slice(samples, func(i, j int) bool {
		// TODO need to take labels into account
		stacktrace1 := samples[i].Location
		stacktrace2 := samples[j].Location

		stacktrace1Len := len(stacktrace1)
		stacktrace2Len := len(stacktrace2)

		k := 1
		for {
			if k == stacktrace1Len && k <= stacktrace2Len {
				// This means the stacktraces are identical up until this point, but stacktrace1 is ending, and shorter stactraces are "lower" than longer ones.
				return true
			}
			if k <= stacktrace1Len && k == stacktrace2Len {
				// This means the stacktraces are identical up until this point, but stacktrace2 is ending, and shorter stactraces are "lower" than longer ones.
				return false
			}
			if stacktrace1[stacktrace1Len-k].ID < stacktrace2[stacktrace2Len-k].ID {
				return true
			}
			if stacktrace1[stacktrace1Len-k].ID > stacktrace2[stacktrace2Len-k].ID {
				return false
			}

			// This means the stack traces are identical up until this point. So advance to the next.
			k++
		}
	})
}
