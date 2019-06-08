export GO111MODULE=on

clean:
	rm -rf .build
	rm -f coverage.txt

stylecheck: clean
	go vet ./...
	gofmt -l -s . | read; if [ $$? == 0 ]; then echo "gofmt check failed for:"; gofmt -d -s .; exit 1; fi
	go get golang.org/x/lint/golint
	golint -set_exit_status ./...

build-init:
	mkdir -p .build/

build-linux: build-init
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o ".build/bumpversion-${TRAVIS_TAG}-linux-amd64"
	gzip ".build/bumpversion-${TRAVIS_TAG}-linux-amd64"

build-mac: build-init
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o ".build/bumpversion-${TRAVIS_TAG}-darwin-amd64"
	gzip ".build/bumpversion-${TRAVIS_TAG}-darwin-amd64"

build-win: build-init
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o ".build/bumpversion-${TRAVIS_TAG}-windows-amd64.exe"
	gzip ".build/bumpversion-${TRAVIS_TAG}-windows-amd64.exe"

build-all: clean build-linux build-mac build-win

test: clean
	go test -coverprofile=coverage.txt -covermode=atomic
