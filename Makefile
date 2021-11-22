OK_COLOR=\033[32;01m
NO_COLOR=\033[0m

build:
	@echo "$(OK_COLOR)==> Compiling binary$(NO_COLOR)"
	go test && go build -o bin/imagor ./cmd/imagor/main.go

test:
	go test

run:
	./bin/imagor

install:
	go get -u .
