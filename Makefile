build:
	CGO_CFLAGS_ALLOW=-Xpreprocessor go build -o bin/imagor ./cmd/imagor/main.go

test:
	go clean -testcache && CGO_CFLAGS_ALLOW=-Xpreprocessor go test -coverprofile=profile.cov ./...

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

%-tag: VERSION:=$(if $(VERSION),$(VERSION),$$(./bin/imagor -version))

git-tag:
	git tag "v$(VERSION)"
	git push --tags

release: build git-tag