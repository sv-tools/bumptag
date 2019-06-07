export GO111MODULE=on

clean:
	rm -rf .build

stylecheck:
	go get golang.org/x/lint/golint
	go vet ./...
	gofmt -l -s . | read; if [ $$? == 0 ]; then echo "gofmt check failed for:"; gofmt -d -s .; exit 1; fi
	golint -set_exit_status ./...

init:
	mkdir -p .build/

build-linux: init
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o ".build/bumpversion-${TRAVIS_TAG}-linux-amd64"
	gzip ".build/bumpversion-${TRAVIS_TAG}-linux-amd64"

build-mac: init
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o ".build/bumpversion-${TRAVIS_TAG}-darwin-amd64"
	gzip ".build/bumpversion-${TRAVIS_TAG}-darwin-amd64"

build-all: clean build-linux build-mac
