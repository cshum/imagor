package vipsprocessor

import (
	"testing"

	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
)

func TestAngleDegrees(t *testing.T) {
	tests := []struct {
		angle vips.Angle
		want  float64
	}{
		{vips.AngleD0, 0},
		{vips.AngleD90, 90},
		{vips.AngleD180, 180},
		{vips.AngleD270, 270},
		{vips.Angle(9), 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, angleDegrees(tt.angle))
	}
}
