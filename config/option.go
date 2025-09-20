package config

import (
	"flag"

	"github.com/cshum/imagor"
	"go.uber.org/zap"
)

// Option flag based config option
type Option func(fs *flag.FlagSet, cb func() (logger *zap.Logger, isDebug bool)) imagor.Option

// applyOptions transform from config.Option to imagor.Option
func applyOptions(
	fs *flag.FlagSet, cb func() (*zap.Logger, bool), options ...Option,
) (imagorOptions []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(options) == 0 {
		logger, isDebug = cb()
		return
	}
	var last = len(options) - 1
	var called bool
	if options[last] == nil {
		return applyOptions(fs, cb, options[:last]...)
	}
	imagorOptions = append(imagorOptions, options[last](fs, func() (*zap.Logger, bool) {
		imagorOptions, logger, isDebug = applyOptions(fs, cb, options[:last]...)
		called = true
		return logger, isDebug
	}))
	if !called {
		var opts []imagor.Option
		opts, logger, isDebug = applyOptions(fs, cb, options[:last]...)
		imagorOptions = append(opts, imagorOptions...)
		return imagorOptions, logger, isDebug
	}
	return
}
