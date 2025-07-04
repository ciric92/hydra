package hydra

import "github.com/spf13/viper"

type options struct {
	supportedExtensions []string
	paths               []string
	viper               *viper.Viper
}

type Option func(*options)

// WithExtensions sets the config file extensions hydra should support.
func WithExtensions(exts ...string) Option {

	return func(o *options) {
		o.supportedExtensions = exts
	}
}

// WithPaths specifies list of files or directories hydra should look for configs in.
func WithPaths(paths ...string) Option {
	return func(o *options) {
		o.paths = paths
	}
}

// WithViper makes hydra use existing viper instance instead of creating a new one.
func WithViper(v *viper.Viper) Option {
	return func(o *options) {
		o.viper = v
	}
}
