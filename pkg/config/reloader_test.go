// Copyright 2022-2023 The Parca Authors
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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func setupReloader(ctx context.Context, t *testing.T) (*os.File, chan *Config) {
	t.Helper()

	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	reloadConfig := make(chan *Config, 1)

	filename := filepath.Join(t.TempDir(), "parca.yaml")

	config := `object_storage:
  bucket:
    type: "FILESYSTEM"
    config:
      directory: "./tmp"

scrape_configs:
  - job_name: "default"
    scrape_interval: "3s"
    static_configs:
      - targets: [ '127.0.0.1:7070' ]
`

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Errorf("failed to open temporary config file: %v", err)
	}

	if _, err := f.WriteString(config); err != nil {
		t.Errorf("failed to write temporary config file: %v", err)
	}

	reloaders := []ComponentReloader{
		{
			Name: "test",
			Reloader: func(cfg *Config) error {
				reloadConfig <- cfg
				return nil
			},
		},
	}

	cfgReloader, err := NewConfigReloader(logger, reg, filename, reloaders)
	if err != nil {
		t.Errorf("failed to instantiate config reloader: %v", err)
	}

	go cfgReloader.Run(ctx)

	time.Sleep(time.Millisecond * 100)

	return f, reloadConfig
}

func TestReloadValid(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*300))
	defer cancel()

	f, reloadConfig := setupReloader(ctx, t)
	defer f.Close()

	config := `    scrape_timeout: "4s"
`

	if _, err := f.WriteString(config); err != nil {
		t.Errorf("failed to update temporary config file: %v", err)
	}

	select {
	case cfg := <-reloadConfig:
		require.Equal(t, model.Duration(time.Second*4), cfg.ScrapeConfigs[0].ScrapeTimeout)
	case <-ctx.Done():
		t.Error("configuration reload timed out")
	}
}

func TestReloadInvalid(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*300))
	defer cancel()

	f, reloadConfig := setupReloader(ctx, t)
	defer f.Close()

	config := "{"

	if _, err := f.WriteString(config); err != nil {
		t.Errorf("failed to update temporary config file: %v", err)
	}

	select {
	case <-reloadConfig:
		t.Error("invalid configuration was reloaded")
	case <-ctx.Done():
	}
}

func TestReloadSymlink(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*300))
	defer cancel()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	reloadConfig := make(chan *Config, 1)

	tmpDir := t.TempDir()
	filenameOld := filepath.Join(tmpDir, "parca_old.yaml")
	filenameNew := filepath.Join(tmpDir, "parca_new.yaml")
	symlinkName := filepath.Join(tmpDir, "parca.yaml")

	config := `object_storage:
  bucket:
    type: "FILESYSTEM"
    config:
      directory: "./tmp"

scrape_configs:
  - job_name: "default"
    scrape_interval: "3s"
    static_configs:
      - targets: [ '127.0.0.1:7070' ]
`

	// Create old config file
	fold, err := os.OpenFile(filenameOld, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Errorf("failed to open config file: %v", err)
	}
	if _, err := fold.WriteString(config); err != nil {
		t.Errorf("failed to write old config file: %v", err)
	}
	fold.Close()

	// Create symlink to old config file
	if err := os.Symlink(filenameOld, symlinkName); err != nil {
		t.Errorf("failed to create symlink to old config file: %v", err)
	}

	config += `    scrape_timeout: "4s"
`

	// Create new config file
	fnew, err := os.OpenFile(filenameNew, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Errorf("failed to open new config file: %v", err)
	}
	if _, err := fnew.WriteString(config); err != nil {
		t.Errorf("failed to write new config file: %v", err)
	}
	fnew.Close()

	// Set up reloader
	reloaders := []ComponentReloader{
		{
			Name: "test",
			Reloader: func(cfg *Config) error {
				reloadConfig <- cfg
				return nil
			},
		},
	}

	cfgReloader, err := NewConfigReloader(logger, reg, symlinkName, reloaders)
	if err != nil {
		t.Errorf("failed to instantiate config reloader: %v", err)
	}

	go cfgReloader.Run(ctx)

	time.Sleep(time.Millisecond * 100)

	// Recreate symlink, but pointing to new config file
	if err := os.Remove(symlinkName); err != nil {
		t.Errorf("failed to remove symlink to old config file: %v", err)
	}
	if err := os.Symlink(filenameNew, symlinkName); err != nil {
		t.Errorf("failed to create symlink to new config file: %v", err)
	}
	// Delete old config file
	// Actually triggers the reload since the symlink was followed
	// when the watcher was created
	// https://github.com/fsnotify/fsnotify/issues/199
	// https://github.com/fsnotify/fsnotify/issues/394
	if err := os.Remove(filenameOld); err != nil {
		t.Errorf("failed to remove old config file: %v", err)
	}

	// Wait for reload
	select {
	case cfg := <-reloadConfig:
		require.Equal(t, model.Duration(time.Second*4), cfg.ScrapeConfigs[0].ScrapeTimeout)
	case <-ctx.Done():
		t.Error("configuration reload timed out")
	}
}
