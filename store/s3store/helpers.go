package s3store

import (
	"path"
	"strings"
)

func Zip(a1, a2 []string) []string {
	r := make([]string, 2*len(a1))
	for i, e := range a1 {
		r[i*2] = e
		r[i*2+1] = a2[i]
	}
	return r
}

func S3Encoder(str string) string {
	array1 := []string{
		"+",
		"\"",
		"#",
		"$",
		"&",
		",",
		":",
		";",
		"=",
		"?",
		"@",
	}
	array2 := []string{
		"%2B",
		"%22",
		"%23",
		"%24",
		"%26",
		"%2C",
		"%3A",
		"%3B",
		"%3D",
		"%3F",
		"%40",
	}
	str = strings.NewReplacer(Zip(array1, array2)...).Replace(str)
	return str
}

// Normalize imagor path to be file path friendly
func S3Normalize(image string) string {
	var escaped []string
	image = path.Clean(image)
	image = strings.Trim(image, "/")
	parts := strings.Split(image, "/")
	for _, part := range parts {
		escaped = append(escaped, S3Encoder(part))
	}
	return strings.Join(escaped, "/")
}
