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

package config

import (
	"context"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// ComponentReloader describes how to reload a component.
type ComponentReloader struct {
	Name     string
	Reloader func(*Config) error
}

// ConfigReloader holds all information required to reload Parca's config into its running components.
type ConfigReloader struct {
	logger            log.Logger
	filename          string
	watcher           *fsnotify.Watcher
	reloaders         []ComponentReloader
	triggerReload     chan struct{}
	configSuccess     prometheus.Gauge
	configSuccessTime prometheus.Gauge
}

// NewConfigReloader returns an instantiated config reloader.
func NewConfigReloader(
	logger log.Logger,
	reg prometheus.Registerer,
	filename string,
	reloaders []ComponentReloader,
) (*ConfigReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		level.Error(logger).Log("msg", "failed to establish config watcher", "err", err)
		return nil, err
	}

	if err := watcher.Add(filename); err != nil {
		level.Error(logger).Log("msg", "failed to start watching config file", "err", err, "path", filename)
		return nil, err
	}

	r := &ConfigReloader{
		logger:   logger,
		filename: filename,
		watcher:  watcher,

		reloaders: reloaders,

		triggerReload: make(chan struct{}, 1),

		configSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "parca_config_last_reload_successful",
			Help: "Whether the last configuration reload attempt was successful.",
		}),
		configSuccessTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "parca_config_last_reload_success_timestamp_seconds",
			Help: "Timestamp of the last successful configuration reload.",
		}),
	}

	if err := reg.Register(r.configSuccess); err != nil {
		return r, fmt.Errorf("unable to register config reloader success metrics: %w", err)
	}

	if err := reg.Register(r.configSuccessTime); err != nil {
		return r, fmt.Errorf("unable to register config reloader success time metrics: %w", err)
	}

	return r, nil
}

func (r *ConfigReloader) watchFile() {
	for {
		select {
		case event, ok := <-r.watcher.Events:
			if !ok {
				level.Debug(r.logger).Log("msg", "config file watcher events channel closed. exiting goroutine.")
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				level.Debug(r.logger).Log("msg", "config file has been modified")
				r.triggerReload <- struct{}{}
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				level.Debug(r.logger).Log("msg", "config file watcher errors channel closed. exiting goroutine.")
				return
			}
			level.Error(r.logger).Log("msg", "error encountered while watching config file", "err", err)
		}
	}
}

func (r *ConfigReloader) reloadFile() (err error) {
	start := time.Now()
	timings := []interface{}{}
	level.Info(r.logger).Log("msg", "loading configuration file", "filename", r.filename)

	defer func() {
		if err == nil {
			r.configSuccess.Set(1)
			r.configSuccessTime.SetToCurrentTime()
		} else {
			r.configSuccess.Set(0)
		}
	}()

	cfg, err := LoadFile(r.filename)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("parsed configuration invalid (--config-path=%q): %w", r.filename, err)
	}

	failed := false
	for _, rl := range r.reloaders {
		rstart := time.Now()
		if err := rl.Reloader(cfg); err != nil {
			level.Error(r.logger).Log("msg", "failed to apply configuration", "err", err)
			failed = true
		}
		timings = append(timings, rl.Name, time.Since(rstart))
	}
	if failed {
		return fmt.Errorf("one or more errors occurred while applying the new configuration (--config-path=%q)", r.filename)
	}

	l := []interface{}{"msg", "completed loading of configuration file", "filename", r.filename, "totalDuration", time.Since(start)}
	level.Info(r.logger).Log(append(l, timings...)...)
	return nil
}

// Run starts watching the config file and wait for reload triggers.
func (r *ConfigReloader) Run(ctx context.Context) error {
	go r.watchFile()
	for {
		select {
		case <-r.triggerReload:
			if err := r.reloadFile(); err != nil {
				level.Error(r.logger).Log("msg", "failed to reload configuration file", "err", err)
			}
		case <-ctx.Done():
			r.watcher.Close()
			return nil
		}
	}
}
