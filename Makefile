export GO111MODULE=on
BUILD_DIR=.build
GPG_KEY=A4BF6A79FB17F115

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
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o "$(BUILD_DIR)/bumptag-${TRAVIS_TAG}-linux-amd64"

build-mac: build-init
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o "$(BUILD_DIR)/bumptag-${TRAVIS_TAG}-darwin-amd64"

build-win: build-init
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o "$(BUILD_DIR)/bumptag-${TRAVIS_TAG}-windows-amd64.exe"

gzip-all:
	find $(BUILD_DIR) -type f -exec gzip {} \;

sign-all:
	find $(BUILD_DIR) -type f -exec gpg -v --default-key $(GPG_KEY) --batch --passphrase "${BUMPTAG_PASSPHRASE}" --output "{}.sig" --detach-sig "{}" \;

build-all: clean build-linux build-mac build-win gzip-all sign-all

test: clean
	go test -coverprofile=coverage.txt -covermode=atomic
