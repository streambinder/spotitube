ROOT_DIR	:= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
MAKE_DIR	:= $(ROOT_DIR)/.make
BUILD_DIR	:= $(ROOT_DIR)/build
BUILD_NAME	:= spotitube
BUILD_LDFLAGS	:= -s -w
BUILD_VERSION	:= $(shell awk -F"= " '/\s+version\s+=/ {print $$2}' main.go)

export MAKE_DIR
export BUILD_DIR
export BUILD_NAME
export BUILD_VERSION

.PHONY: build
build: build-linux build-windows

.PHONY: build-linux
build-linux:
	@GOARCH=386 CGO_ENABLED=0 go build -ldflags="$(BUILD_LDFLAGS)" -o $(BUILD_DIR)/$(BUILD_NAME)

.PHONY: build-windows
build-windows:
	@GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -ldflags="$(BUILD_LDFLAGS)" -o $(BUILD_DIR)/$(BUILD_NAME).exe

.PHONY: install
install: build
	@go install

.PHONY: release
release: keys-inflate build package keys-deflate
	@cp $(BUILD_DIR)/$(BUILD_NAME) $(BUILD_DIR)/$(BUILD_NAME)-v${BUILD_VERSION}.bin
	@cp $(BUILD_DIR)/$(BUILD_NAME).exe $(BUILD_DIR)/$(BUILD_NAME)-v${BUILD_VERSION}.exe

.PHONY: keys-inflate
keys-inflate:
	@bash $(MAKE_DIR)/keys-inflate.sh

.PHONY: keys-deflate
keys-deflate:
	@bash $(MAKE_DIR)/keys-deflate.sh

.PHONY: package
package: package-deb package-rpm package-eopkg

.PHONY: package-deb
package-deb: build-linux
	@bash $(MAKE_DIR)/package-deb.sh

.PHONY: package-rpm
package-rpm: build-linux
	@bash $(MAKE_DIR)/package-rpm.sh

.PHONY: package-eopkg
package-eopkg: build-linux
	@bash $(MAKE_DIR)/package-eopkg.sh

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)