package vipsprocessor

type Focal struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func ParseFocalPoint(width, height int, focalRects ...Focal) (focalX, focalY float64) {
	var sumWeight float64
	var dw = float64(width)
	var dh = float64(height)
	for _, f := range focalRects {
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
