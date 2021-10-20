package symbol

import (
	"time"

	"github.com/goburrow/cache"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

type Option func(*Symbolizer)

func WithAttemptThreshold(t int) Option {
	return func(s *Symbolizer) {
		s.attemptThreshold = t
	}
}

func WithDemangleMode(mode string) Option {
	return func(s *Symbolizer) {
		s.demangler = demangle.NewDemangler(mode, false)
	}
}

func WithCacheSize(size int) Option {
	return func(s *Symbolizer) {
		s.cacheOpts = append(s.cacheOpts, cache.WithMaximumSize(size))
	}
}

func WithCacheItemTTL(ttl time.Duration) Option {
	return func(s *Symbolizer) {
		s.cacheOpts = append(s.cacheOpts, cache.WithExpireAfterAccess(ttl))
	}
}
