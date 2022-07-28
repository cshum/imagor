package config

import (
	"flag"
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestCreateOptions(t *testing.T) {
	fs := flag.NewFlagSet("imagaor", flag.ExitOnError)
	nopLogger := zap.NewNop()
	op1 := func(app *imagor.Imagor) {}
	op2 := func(app *imagor.Imagor) {}
	op3 := func(app *imagor.Imagor) {}
	var seq []int

	ApplySetters(fs, func() (logger *zap.Logger, isDebug bool) {
		seq = append(seq, 4)
		return nopLogger, true
	}, func(fs *flag.FlagSet, cb Callback) imagor.Option {
		seq = append(seq, 3)
		logger, isDebug := cb()
		assert.Equal(t, nopLogger, logger)
		assert.True(t, isDebug)
		seq = append(seq, 5)
		return op1
	}, func(fs *flag.FlagSet, cb Callback) imagor.Option {
		seq = append(seq, 2)
		logger, isDebug := cb()
		assert.Equal(t, nopLogger, logger)
		assert.True(t, isDebug)
		seq = append(seq, 6)
		return op2
	}, func(fs *flag.FlagSet, cb Callback) imagor.Option {
		seq = append(seq, 1)
		logger, isDebug := cb()
		assert.Equal(t, nopLogger, logger)
		assert.True(t, isDebug)
		seq = append(seq, 7)
		return op3
	})
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7}, seq)
}
