# imagorpath

Parse and generate Imagor endpoint using Go struct

```go
import "github.com/cshum/imagor/imagorpath"

...

func Test(t *testing.T) {
	params := imagorpath.Params{
		Image:    "raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
		FitIn:    true,
		Width:    500,
		Height:   400,
		VPadding: 20,
		Filters: imagorpath.Filters{
			{
				Name: "fill",
				Args: "white",
			},
		},
	}

	// generate signed Imagor endpoint from Params struct with secret
	path := imagorpath.Generate(params, "mysecret")

	assert.Equal(t, path, "g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png")

	assert.Equal(t,
		// parse Params struct from signed Imagor endpoint
		imagorpath.Parse(path),

		// Params include endpoint attributes with path and signed hash
		imagorpath.Params{
			Path:     "fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
			Hash:     "g5bMqZvxaQK65qFPaP1qlJOTuLM=",
			Image:    "raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
			FitIn:    true,
			Width:    500,
			Height:   400,
			VPadding: 20,
			Filters: imagorpath.Filters{
				{
					Name: "fill",
					Args: "white",
				},
			},
		},
	)
}

```