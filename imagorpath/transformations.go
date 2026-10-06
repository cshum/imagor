package imagorpath

// HasTransformations reports whether the params ask imagor to do anything to the
// image: a resize, fit, stretch, alignment, padding, flip, crop, trim, smart
// crop, or any filter that is not in ignoreFilters.
//
// ignoreFilters lets a caller skip filters it has already accounted for - a
// format the server negotiated itself, or its own marker - so a caller that
// cannot tell a client-named filter from a synthetic one does not have to guess.
func HasTransformations(p Params, ignoreFilters ...string) bool {
	if hasGeometryOps(p) {
		return true
	}
	for _, f := range p.Filters {
		if isIgnoredFilter(f.Name, ignoreFilters) {
			continue
		}
		return true
	}
	return false
}

func isIgnoredFilter(name string, ignoreFilters []string) bool {
	for _, ignore := range ignoreFilters {
		if name == ignore {
			return true
		}
	}
	return false
}

// hasGeometryOps reports whether the params carry a sizing, positioning or crop
// instruction - what makes a request an operation rather than a request for the
// source as it is.
func hasGeometryOps(p Params) bool {
	return p.Meta || p.Trim || p.TrimBy != "" || p.TrimTolerance != 0 ||
		HasCrop(p) || p.FitIn || p.AdaptiveFitIn || p.FullFitIn || p.Stretch ||
		p.Width != 0 || p.Height != 0 ||
		p.PaddingLeft != 0 || p.PaddingTop != 0 ||
		p.PaddingRight != 0 || p.PaddingBottom != 0 ||
		p.HFlip || p.VFlip || p.HAlign != "" || p.VAlign != "" || p.Smart
}
