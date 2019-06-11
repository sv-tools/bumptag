export GO111MODULE=on
BUILD_DIR=.build
GPG_KEY=F2DA1B9CE7F25D1B

clean:
	rm -rf .build
	rm -f coverage.txt

stylecheck: clean
	go vet ./...
	gofmt -l -s . | read; if [ $$? == 0 ]; then echo "gofmt check failed for:"; gofmt -d -s .; exit 1; fi
	go get golang.org/x/lint/golint
	golint -set_exit_status ./...

define build_bumptag
	GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.version=${TRAVIS_TAG} -s -w" -o "$(BUILD_DIR)/bumptag-${TRAVIS_TAG}-$(1)-$(2)$(3)"
endef

define gzip_bumptag
	$(eval $@_NAME := bumptag-${TRAVIS_TAG}-$(1)-$(2)$(3))
	mv $(BUILD_DIR)/$($@_NAME) $(BUILD_DIR)/bumptag$(3)
	tar -C $(BUILD_DIR) -czvf $(BUILD_DIR)/$($@_NAME).tgz bumptag$(3)
	rm $(BUILD_DIR)/bumptag$(3)
endef

define sign_bumptag
	$(eval $@_NAME := $(BUILD_DIR)/bumptag-${TRAVIS_TAG}-$(1)-$(2)$(3).tgz)
	gpg --default-key $(GPG_KEY) --batch --passphrase "${BUMPTAG_PASSPHRASE}" --output "$($@_NAME).sig" --detach-sig "$($@_NAME)"
endef

define sha256_bumptag
	$(eval $@_NAME := bumptag-${TRAVIS_TAG}-$(1)-$(2)$(3).tgz)
	cd $(BUILD_DIR) ; \
	sha256sum "$($@_NAME)" > "$($@_NAME).sha256.txt"
endef

build-init:
	mkdir -p .build/

gzip-all:
	$(call gzip_bumptag,linux,amd64)
	$(call gzip_bumptag,darwin,amd64)
	$(call gzip_bumptag,windows,amd64,".exe")

sign-all:
	$(call sign_bumptag,linux,amd64)
	$(call sign_bumptag,darwin,amd64)
	$(call sign_bumptag,windows,amd64,".exe")

sha256-all:
	$(call sha256_bumptag,linux,amd64)
	$(call sha256_bumptag,darwin,amd64)
	$(call sha256_bumptag,windows,amd64,".exe")

build-all: clean build-init
	$(call build_bumptag,linux,amd64)
	$(call build_bumptag,darwin,amd64)
	$(call build_bumptag,windows,amd64,".exe")

prepare-release: build-all gzip-all sign-all sha256-all

test: clean
	go test -coverprofile=coverage.txt -covermode=atomic
