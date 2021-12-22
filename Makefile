build:
	CGO_CFLAGS_ALLOW=.* CGO_LDFLAGS_ALLOW=.* go build -o bin/imagor ./cmd/imagor/main.go

dev: build
	./bin/imagor -debug -imagor-unsafe

test:
	CGO_CFLAGS_ALLOW=.* CGO_LDFLAGS_ALLOW=.* go test -cover ./...

docker-dev-build:
	docker build --build-arg IMAGOR_VERSION=dev -t shumc/imagor:dev .

docker-dev-run:
	touch .env
	docker run -p 8000:8000 --env-file .env shumc/imagor:dev -debug -imagor-unsafe

docker-dev: docker-dev-build docker-dev-run

docker-build:
	docker build --no-cache=true --build-arg IMAGOR_VERSION=$(VERSION) -t shumc/imagor:$(VERSION) .

docker-push:
	docker push shumc/imagor:$(VERSION)

docker-latest:
	docker tag shumc/imagor:$(VERSION) shumc/imagor:latest
	docker push shumc/imagor:latest

docker-tag: docker-build docker-push

docker: docker-build docker-push docker-latest
