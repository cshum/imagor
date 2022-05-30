package vipsprocessor

import "math"

type Focal struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func (f Focal) X() float64 {
	return (f.Left + f.Right) / 2
}

func (f Focal) Y() float64 {
	return (f.Top + f.Bottom) / 2
}

func (f Focal) Weight() float64 {
	return math.Abs(f.Right-f.Left) * math.Abs(f.Bottom-f.Top)
}

func ParseFocalPoint(width, height int, focalRects ...Focal) (focalX, focalY float64) {
	return
}
