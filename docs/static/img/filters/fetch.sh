#!/bin/bash
IMG="raw.githubusercontent.com/cshum/imagor/refs/heads/master/testdata/jpg-24bit-icc-adobe-rgb.jpg"
GOPHER="raw.githubusercontent.com/cshum/imagor/refs/heads/master/testdata/gopher-front.png"
BASE="http://localhost:8000/unsafe"
OUT="/Users/cshum/Developer/imagor/docs/static/img/filters"

curl -s "$BASE/400x400/filters:format(jpeg)/$IMG" -o "$OUT/original.jpg" && echo "original ok"
curl -s "$BASE/400x400/filters:blur(5):format(jpeg)/$IMG" -o "$OUT/blur.jpg" && echo "blur ok"
curl -s "$BASE/400x400/filters:brightness(50):format(jpeg)/$IMG" -o "$OUT/brightness-plus.jpg" && echo "brightness+ ok"
curl -s "$BASE/400x400/filters:brightness(-50):format(jpeg)/$IMG" -o "$OUT/brightness-minus.jpg" && echo "brightness- ok"
curl -s "$BASE/400x400/filters:contrast(50):format(jpeg)/$IMG" -o "$OUT/contrast-plus.jpg" && echo "contrast+ ok"
curl -s "$BASE/400x400/filters:contrast(-50):format(jpeg)/$IMG" -o "$OUT/contrast-minus.jpg" && echo "contrast- ok"
curl -s "$BASE/400x400/filters:grayscale():format(jpeg)/$IMG" -o "$OUT/grayscale.jpg" && echo "grayscale ok"
curl -s "$BASE/400x400/filters:hue(90):format(jpeg)/$IMG" -o "$OUT/hue.jpg" && echo "hue ok"
curl -s "$BASE/400x400/filters:saturation(100):format(jpeg)/$IMG" -o "$OUT/saturation-plus.jpg" && echo "saturation+ ok"
curl -s "$BASE/400x400/filters:saturation(-80):format(jpeg)/$IMG" -o "$OUT/saturation-minus.jpg" && echo "saturation- ok"
curl -s "$BASE/400x400/filters:rgb(60,-30,-30):format(jpeg)/$IMG" -o "$OUT/rgb.jpg" && echo "rgb ok"
curl -s "$BASE/400x400/filters:pixelate(10):format(jpeg)/$IMG" -o "$OUT/pixelate.jpg" && echo "pixelate ok"
curl -s "$BASE/400x400/filters:sharpen(3):format(jpeg)/$IMG" -o "$OUT/sharpen.jpg" && echo "sharpen ok"
curl -s "$BASE/400x400/filters:rotate(90):format(jpeg)/$IMG" -o "$OUT/rotate.jpg" && echo "rotate ok"
curl -s "$BASE/fit-in/400x400/filters:fill(white):round_corner(40):format(jpeg)/$IMG" -o "$OUT/round-corner.jpg" && echo "round-corner ok"
curl -s "$BASE/400x400/filters:watermark($GOPHER,-20,-20,0,30,30):format(jpeg)/$IMG" -o "$OUT/watermark.jpg" && echo "watermark ok"
curl -s "$BASE/400x400/filters:watermark($GOPHER,repeat,bottom,30,30,30):format(jpeg)/$IMG" -o "$OUT/watermark-repeat.jpg" && echo "watermark-repeat ok"
curl -s "$BASE/400x400/filters:quality(5):format(jpeg)/$IMG" -o "$OUT/quality-low.jpg" && echo "quality ok"

# image() filter examples (400x400 base crop)
GOPHER2="raw.githubusercontent.com/cshum/imagor/refs/heads/master/testdata/gopher.png"
curl -s "$BASE/400x400/filters:image(/fit-in/100x100/$GOPHER,center,center):format(jpeg)/$IMG" -o "$OUT/image-center.jpg" && echo "image-center ok"
curl -s "$BASE/400x400/filters:image(/fit-in/100x100/$GOPHER,center,center,50):format(jpeg)/$IMG" -o "$OUT/image-alpha.jpg" && echo "image-alpha ok"
curl -s "$BASE/400x400/filters:image(/fit-in/150x150/filters:image(/fit-in/50x50/$GOPHER,center,center)/$GOPHER2,10,10):format(jpeg)/$IMG" -o "$OUT/image-nested.jpg" && echo "image-nested ok"

# text() filter examples (400x400 base crop, high-contrast text colors)
curl -s "$BASE/400x400/filters:text(IMAGOR,20,20,sans-bold-36,white,0):format(jpeg)/$IMG" -o "$OUT/text-basic.jpg" && echo "text-basic ok"
curl -s "$BASE/400x400/filters:text(b64:SGVsbG8gV29ybGQgZnJvbSBpbWFnb3I,-20,20,sans-24,yellow,0,,180,high):format(jpeg)/$IMG" -o "$OUT/text-multiline.jpg" && echo "text-multiline ok"
