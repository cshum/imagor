build:
	go test && go build -o bin/imagor ./cmd/imagor/main.go

test:
	go test

run:
	./bin/imagor

dev:
	make build && ./bin/imagor -debug

docker-build:
	docker build --no-cache=true --build-arg IMAGOR_VERSION=$(VERSION) -t shumc/imagor:$(VERSION) .

docker-push:
	docker push shumc/imagor:$(VERSION)

docker: docker-build docker-push

install:
	go get -u .