package imagorpath

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasTransformations(t *testing.T) {
	noop := Params{Image: "plain/photo.svg"}
	assert.False(t, HasTransformations(noop))

	for _, tc := range []struct {
		name   string
		params Params
	}{
		{"meta", Params{Image: "x", Meta: true}},
		{"width", Params{Image: "x", Width: 1}},
		{"height", Params{Image: "x", Height: 1}},
		{"fit-in", Params{Image: "x", FitIn: true}},
		{"adaptive fit-in", Params{Image: "x", AdaptiveFitIn: true}},
		{"full fit-in", Params{Image: "x", FullFitIn: true}},
		{"stretch", Params{Image: "x", Stretch: true}},
		{"smart", Params{Image: "x", Smart: true}},
		{"trim", Params{Image: "x", Trim: true}},
		{"trim by", Params{Image: "x", TrimBy: TrimByTopLeft}},
		{"trim tolerance", Params{Image: "x", TrimTolerance: 1}},
		{"crop", Params{Image: "x", CropLeft: 0.1}},
		{"padding", Params{Image: "x", PaddingTop: 1}},
		{"hflip", Params{Image: "x", HFlip: true}},
		{"vflip", Params{Image: "x", VFlip: true}},
		{"h align", Params{Image: "x", HAlign: HAlignLeft}},
		{"v align", Params{Image: "x", VAlign: VAlignTop}},
		{"any filter", Params{Image: "x", Filters: Filters{{Name: "blur", Args: "2"}}}},
		{"format filter", Params{Image: "x", Filters: Filters{{Name: "format", Args: "webp"}}}},
		{"raw filter", Params{Image: "x", Filters: Filters{{Name: "raw"}}}},
		{"preview filter", Params{Image: "x", Filters: Filters{{Name: "preview"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, HasTransformations(tc.params), "expected an operation")
		})
	}
}

func TestHasTransformationsIgnoreFilters(t *testing.T) {
	ignore := []string{"format", "fallback_format", "autojpg", "passthrough"}

	// A filter a caller has accounted for does not, on its own, make the
	// request an operation.
	for _, name := range ignore {
		p := Params{Image: "x", Filters: Filters{{Name: name}}}
		assert.False(t, HasTransformations(p, ignore...), "filter %q", name)
		assert.True(t, HasTransformations(p), "filter %q without the ignore list", name)
	}

	// Anything else still does, and geometry always does.
	p := Params{Image: "x", Filters: Filters{{Name: "format"}, {Name: "rotate", Args: "90"}}}
	assert.True(t, HasTransformations(p, ignore...))

	p = Params{Image: "x", Width: 1, Filters: Filters{{Name: "format"}}}
	assert.True(t, HasTransformations(p, ignore...))
}

// TestHasTransformationsCoversEveryParamsField is a maintenance tripwire: a
// field added to Params must be considered here, either as something that
// describes the request itself (and may be set freely) or as something that asks
// for work (and must report an operation).
//
// Without it, a new sizing or crop field could silently fail to count as an
// operation, and a request carrying it could be answered with the untouched
// source image.
func TestHasTransformationsCoversEveryParamsField(t *testing.T) {
	// Fields that describe the request rather than ask for work.
	metadata := map[string]bool{
		"Params": true, "Path": true, "Image": true, "Base64Image": true,
		"Unsafe": true, "Hash": true,
	}
	paramsType := reflect.TypeOf(Params{})
	for i := 0; i < paramsType.NumField(); i++ {
		field := paramsType.Field(i)
		if metadata[field.Name] {
			continue
		}
		p := Params{Image: "x"}
		v := reflect.ValueOf(&p).Elem().Field(i)
		switch v.Kind() {
		case reflect.Bool:
			v.SetBool(true)
		case reflect.Int:
			v.SetInt(1)
		case reflect.Float64:
			v.SetFloat(1)
		case reflect.String:
			v.SetString("set")
		case reflect.Slice:
			v.Set(reflect.ValueOf(Filters{{Name: "f"}}))
		default:
			t.Fatalf("field %s of kind %s is not covered by this test", field.Name, v.Kind())
		}
		assert.Truef(t, HasTransformations(p),
			"Params.%s does not count as an operation: a request setting it could be "+
				"mistaken for a no-op", field.Name)
	}

	// The metadata fields must not affect the result.
	for name := range metadata {
		p := Params{Image: "x"}
		v := reflect.ValueOf(&p).Elem().FieldByName(name)
		switch v.Kind() {
		case reflect.Bool:
			v.SetBool(true)
		case reflect.String:
			v.SetString("set")
		}
		assert.Falsef(t, HasTransformations(p), "%s must not count as an operation", name)
	}
}
