package s3store

import (
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
	array1 := []string { 
		"+",
		"!",
		"\"",
		"#",
		"$",
		"&",
		"'",
		"(",
		")",
		"*",
		",",
		":",
		";",
		"=",
		"?",
		"@",
	}
	array2 := []string { 
		"%2B",
		"%21",
		"%22",
		"%23",
		"%24",
		"%26",
		"%27",
		"%28",
		"%29",
		"%2A",
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