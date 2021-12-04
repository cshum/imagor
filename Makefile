build:
	go build -o bin/imagor ./cmd/imagor/main.go

test:
	go test

dev: build
	./bin/imagor -debug

docker-build:
	docker build --no-cache=true --build-arg IMAGOR_VERSION=$(VERSION) -t shumc/imagor:$(VERSION) .

docker-push:
	docker tag shumc/imagor:$(VERSION) shumc/imagor:latest
	docker push shumc/imagor:$(VERSION)
	docker push shumc/imagor:latest

docker: docker-build docker-push

install:
	go get -u .