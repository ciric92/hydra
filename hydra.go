package hydra

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Hydra extends Viper's functionality by adding support for watching and loading multiple
// configuration files.
//
// It supports:
// 1. Recursivelly searching directories for configuration files
// 2. Single configuration files
// 3. Symlinks
type Hydra struct {
	viper       *viper.Viper
	watcher     *fsnotify.Watcher
	options     *options
	configFiles []string
}

// New creates a new hydra instance.
func New(opts ...Option) (*Hydra, error) {

	o := options{
		supportedExtensions: viper.SupportedExts,
		paths:               []string{"."},
	}
	for _, opt := range opts {
		opt(&o)
	}

	if o.viper == nil {
		o.viper = viper.New()
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create a new watcher: %w", err)
	}

	h := Hydra{
		viper:   o.viper,
		watcher: w,
		options: &o,
	}

	for _, path := range o.paths {
		err := h.addPath(path)
		if err != nil {
			return nil, fmt.Errorf("add path (path: %s): %w", path, err)
		}
	}

	return &h, nil
}

// Start starts watching for changes in the configuration.
func (h *Hydra) Start(ctx context.Context, notify NotifyFunc) error {
	for {
		select {
		case ev, ok := <-h.watcher.Events:
			if !ok {
				return errors.New("watcher unexpectedly closed")
			}

			ext := strings.TrimPrefix(filepath.Ext(ev.Name), ".")
			if !slices.Contains(h.options.supportedExtensions, ext) {
				// file extension is not supported, so no config is loaded
				continue
			}

			if ev.Op&(fsnotify.Remove|fsnotify.Create|fsnotify.Rename|fsnotify.Write) == 0 {
				// operation does not trigger the file change
				continue
			}

			notify(ev.Name, ev.Op)
		case <-ctx.Done():
			err := h.watcher.Close()
			if err != nil {
				return fmt.Errorf("close watcher: %w", err)
			}
			return nil
		}
	}
}

// ConfigFiles returns paths to loaded configuration files.
func (h *Hydra) ConfigFiles() []string {
	return h.configFiles
}

func (h *Hydra) addPath(path string) error {
	h.watcher.Add(path)
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// watching isn't recursive so the path needs to be added to the watcher.
			h.watcher.Add(path)
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		if !slices.Contains(h.options.supportedExtensions, ext) {
			// file extension is not supported
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// if config file is symlink then read the real path
			path, err = os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read config file link (path: %s): %w", path, err)
			}
		}

		// config file found
		h.configFiles = append(h.configFiles, path)
		firstConfigFile := h.viper.ConfigFileUsed() == ""
		h.viper.SetConfigFile(path)

		if firstConfigFile {
			err := h.viper.ReadInConfig()
			if err != nil {
				return fmt.Errorf("read in config file (path: %s): %w", path, err)
			}
			return nil
		}

		err = h.viper.MergeInConfig()
		if err != nil {
			return fmt.Errorf("merge in config file (path: %s): %w", path, err)
		}

		return nil
	})
}
