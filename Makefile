build:
	CGO_CFLAGS_ALLOW=-Xpreprocessor go build -o bin/imagor ./cmd/imagor/main.go

test:
	go clean -testcache && CGO_CFLAGS_ALLOW=-Xpreprocessor go test -cover ./...

dev: build
	./bin/imagor -debug -imagor-unsafe

help: build
	./bin/imagor -h

get:
	go get -v -t -d ./...

docker-dev-build:
	docker build -t shumc/imagor:dev .

docker-dev-run:
	touch .env
	docker run -p 8000:8000 --env-file .env shumc/imagor:dev -debug -imagor-unsafe

docker-dev: docker-dev-build docker-dev-run

%-tag: VERSION:=$(if $(VERSION),$(VERSION),$$(./bin/imagor -version))

docker-build-tag:
	docker build --no-cache=true -t shumc/imagor:$(VERSION) .

docker-push-tag:
	docker push shumc/imagor:$(VERSION)

docker-latest-tag:
	docker tag shumc/imagor:$(VERSION) shumc/imagor:latest
	docker push shumc/imagor:latest

docker-tag: docker-build-tag docker-push-tag

git-tag:
	git tag "v$(VERSION)"
	git push --tags

release: test build docker-tag git-tag docker-latest-tag