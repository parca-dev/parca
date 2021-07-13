package chunk

import (
	"bytes"
)

type Chunk struct {
	// A slice of samples for each unique stack trace.
	stacktraces []Stacktrace
	timestamps  []int64
	durations   []int64
	periods     []int64
}

type DecodedData struct {
	Stacktraces []Stacktrace
	Timestamps  []int64
	Durations   []int64
	Periods     []int64
}

type Sample struct {
	Timestamp   int64
	Duration    int64
	Period      int64
	Stacktraces []StacktraceSample
}

type StacktraceSample struct {
	StacktraceID [16]byte
	Value        int64
}

type Stacktrace struct {
	StacktraceID [16]byte
	// Unlike the pprof Profile, this doesn't contain different values, but rather a series of values.
	Values []int64
}

// Appends the stacktrace samples to the respective stacktrace sample chunks.
// Note: The stacktraces must be ordered by their ID.
func (c *Chunk) Append(s Sample) error {
	c.timestamps = append(c.timestamps, s.Timestamp)
	c.durations = append(c.durations, s.Duration)
	c.periods = append(c.periods, s.Period)

	if len(s.Stacktraces) == 0 {
		return nil
	}

	if len(c.stacktraces) == 0 {
		c.stacktraces = make([]Stacktrace, 0, len(s.Stacktraces))
		for _, stacktraceSample := range s.Stacktraces {
			c.stacktraces = append(c.stacktraces, Stacktrace{
				StacktraceID: stacktraceSample.StacktraceID,
				Values:       []int64{stacktraceSample.Value},
			})
		}
		return nil
	}

	i, j := 0, 0
	for j < len(s.Stacktraces) && i < len(c.stacktraces) {
		cmp := bytes.Compare(c.stacktraces[i].StacktraceID[:], s.Stacktraces[j].StacktraceID[:])

		if cmp == -1 {
			c.stacktraces[i].Values = append(c.stacktraces[i].Values, 0)
			i++
			continue
		}

		// This means the next known stacktrace is larger than the one appended to.
		// It means we don't know about this stacktrace yet so we need to insert it.
		if cmp == 1 {
			// TODO: benchmark "insert into slice" vs. a linked list as well as
			// just appending unknown ones and sorting. After naively
			// implementing this I think sorting should be best.
			newStacktraces := make([]Stacktrace, len(c.stacktraces)+1)
			copy(newStacktraces, c.stacktraces[:i])
			newStacktraces[i] = Stacktrace{
				StacktraceID: s.Stacktraces[j].StacktraceID,
				Values:       make([]int64, len(c.timestamps)-1),
			}
			copy(newStacktraces[i+1:], c.stacktraces[i:])
			c.stacktraces = newStacktraces
		}

		c.stacktraces[i].Values = append(c.stacktraces[i].Values, s.Stacktraces[j].Value)
		i++
		j++
	}

	if j < len(s.Stacktraces) {
		for j < len(s.Stacktraces) {
			// All stacktraces left in the sample are so far unknown ones.
			v := make([]int64, len(c.timestamps))
			v[len(v)-1] = s.Stacktraces[j].Value
			c.stacktraces = append(c.stacktraces, Stacktrace{
				StacktraceID: s.Stacktraces[j].StacktraceID,
				Values:       v,
			})
			j++
		}
		return nil
	}

	for i < len(c.stacktraces) {
		c.stacktraces[i].Values = append(c.stacktraces[i].Values, 0)
		i++
	}

	return nil
}

// TODO: Once the data is encoded in the Chunk, this will need to decode it to
// return it as part of ChunkData. It would also make sense to have hints
// passed to this to pre-select or aggregate data. Examples of possible hints:
// timestamps only, certain stacktrace IDs only, merge window (though it's
// possible that this might make more sense at a level above that also knows
// the entire timestamp range of the chunk).
func (c *Chunk) Data() DecodedData {
	return DecodedData{
		Stacktraces: c.stacktraces,
		Timestamps:  c.timestamps,
		Durations:   c.durations,
		Periods:     c.periods,
	}
}
