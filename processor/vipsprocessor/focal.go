package vipsprocessor

type Focal struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func ParseFocalPoint(width, height int, focalRects ...*Focal) (focalX, focalY float64) {
	var sumWeight float64
	var dw = float64(width)
	var dh = float64(height)
	for _, f := range focalRects {
		if f.Left < 1 && f.Top < 1 && f.Right <= 1 && f.Bottom <= 1 {
			f.Left *= dw
			f.Right *= dw
			f.Top *= dh
			f.Bottom *= dh
		}
		sumWeight += (f.Right - f.Left) * (f.Bottom - f.Top)
	}
	for _, f := range focalRects {
		r := (f.Right - f.Left) * (f.Bottom - f.Top) / sumWeight
		focalX += (f.Left + f.Right) / 2 * r
		focalY += (f.Top + f.Bottom) / 2 * r
	}
	focalX /= dw
	focalY /= dh
	return
}
