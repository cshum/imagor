#!/bin/bash
IMG="raw.githubusercontent.com/cshum/imagor/refs/heads/master/testdata/jpg-24bit-icc-adobe-rgb.jpg"
GOPHER="raw.githubusercontent.com/cshum/imagor/refs/heads/master/testdata/gopher.png"
BASE="http://localhost:8000/unsafe"
OUT="/Users/cshum/Developer/imagor/docs/static/img/endpoint"

# Resize & crop
curl -s "$BASE/400x400/filters:format(jpeg)/$IMG" -o "$OUT/resize-crop.jpg" && echo "resize-crop ok"

# Alignment
curl -s "$BASE/400x400/left/top/filters:format(jpeg)/$IMG" -o "$OUT/align-left-top.jpg" && echo "align-left-top ok"
curl -s "$BASE/400x400/center/middle/filters:format(jpeg)/$IMG" -o "$OUT/align-center.jpg" && echo "align-center ok"
curl -s "$BASE/400x400/right/bottom/filters:format(jpeg)/$IMG" -o "$OUT/align-right-bottom.jpg" && echo "align-right-bottom ok"

# Smart crop
curl -s "$BASE/400x400/smart/filters:format(jpeg)/$IMG" -o "$OUT/smart-crop.jpg" && echo "smart-crop ok"

# Fit-in
curl -s "$BASE/fit-in/400x400/filters:format(jpeg)/$IMG" -o "$OUT/fit-in.jpg" && echo "fit-in ok"
curl -s "$BASE/fit-in/400x400/filters:fill(red):format(jpeg)/$IMG" -o "$OUT/fit-in-fill-red.jpg" && echo "fit-in-fill-red ok"
curl -s "$BASE/fit-in/400x400/filters:fill(blur):format(jpeg)/$IMG" -o "$OUT/fit-in-fill-blur.jpg" && echo "fit-in-fill-blur ok"
curl -s "$BASE/fit-in/400x400/filters:fill(white):format(jpeg)/$IMG" -o "$OUT/fit-in-fill-white.jpg" && echo "fit-in-fill-white ok"
curl -s "$BASE/fit-in/400x400/filters:fill(grey):format(jpeg)/$IMG" -o "$OUT/fit-in-fill-grey.jpg" && echo "fit-in-fill-grey ok"

# Full fit-in & adaptive fit-in
curl -s "$BASE/full-fit-in/400x400/filters:format(jpeg)/$IMG" -o "$OUT/full-fit-in.jpg" && echo "full-fit-in ok"
curl -s "$BASE/adaptive-fit-in/400x400/filters:format(jpeg)/$IMG" -o "$OUT/adaptive-fit-in.jpg" && echo "adaptive-fit-in ok"

# Stretch
curl -s "$BASE/stretch/400x400/filters:format(jpeg)/$IMG" -o "$OUT/stretch.jpg" && echo "stretch ok"

# Flip
curl -s "$BASE/-400x400/filters:format(jpeg)/$IMG" -o "$OUT/flip-h.jpg" && echo "flip-h ok"
curl -s "$BASE/400x-400/filters:format(jpeg)/$IMG" -o "$OUT/flip-v.jpg" && echo "flip-v ok"

# Manual crop
curl -s "$BASE/100x50:1800x1200/400x400/filters:format(jpeg)/$IMG" -o "$OUT/manual-crop.jpg" && echo "manual-crop ok"

# Padding (transparent PNG + fill yellow, padding after size)
curl -s "$BASE/fit-in/360x360/20x20:20x20/filters:fill(yellow):format(jpeg)/$GOPHER" -o "$OUT/padding.jpg" && echo "padding ok"

# Trim
curl -s "$BASE/trim/400x400/filters:format(jpeg)/$IMG" -o "$OUT/trim.jpg" && echo "trim ok"
