.PHONY: build clean deploy

build:
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o bin/hello/bootstrap hello/main.go
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o bin/world/bootstrap world/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
