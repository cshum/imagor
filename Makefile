build:
	CGO_CFLAGS_ALLOW=-Xpreprocessor go build -o bin/imagor ./cmd/imagor/main.go

test:
	go clean -testcache && CGO_CFLAGS_ALLOW=-Xpreprocessor go test -coverprofile=profile.cov $(shell go list ./... | grep -v /examples/ | grep -v /cmd/)

dev: build
	./bin/imagor -debug -imagor-unsafe

help: build
	./bin/imagor -h

get:
	go get -v -t -d ./...

docker-dev-build:
	docker build -t imagor:dev .

docker-dev-run:
	touch .env
	docker run --rm -p 8000:8000 --env-file .env imagor:dev -debug -imagor-unsafe

docker-dev: docker-dev-build docker-dev-run

docker-magick-build:
	docker build --build-arg ENABLE_MAGICK=true -t imagor:magick .

docker-magick-run:
	touch .env
	docker run --rm -p 8000:8000 --env-file .env imagor:magick -debug -imagor-unsafe

docker-magick: docker-magick-build docker-magick-run

docker-mozjpeg-build:
	docker build --build-arg ENABLE_MOZJPEG=true -t imagor:mozjpeg .

docker-mozjpeg-run:
	touch .env
	docker run --rm -p 8000:8000 --env-file .env imagor:mozjpeg -debug -imagor-unsafe -vips-mozjpeg

docker-mozjpeg: docker-mozjpeg-build docker-mozjpeg-run

%-tag: VERSION:=$(if $(VERSION),$(VERSION),$$(./bin/imagor -version))

git-tag:
	git tag "v$(VERSION)"
	git push origin "refs/tags/v$(VERSION)"

reset-golden:
	git rm -rf testdata/golden
	git commit -m  "test: reset golden"
	git push

release: build git-tag
