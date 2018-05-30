.DEFAULT_GOAL := all
PROJECT_ROOT := src/

NAME := spotitube
CUR_DIR = $(shell pwd)
BINARY := $(CUR_DIR)/out/$(NAME)
BINARY_INSTALL_PATH := /usr/local/sbin
BINARY_INSTALL := $(BINARY_INSTALL_PATH)/$(NAME)
VERSION := $(shell awk -F'= ' '/Version / {print $$2}' src/system/constants.go)
PKG_NAME := $(BINARY)-v$(VERSION)
GOARCH := amd64
LDFLAGS := -s -w

include Makefile.gobuild
include Makefile.packaging

.PHONY: install
install: bin
	@ ( \
		echo -en "Installing... "; \
		(test -d $(BINARY_INSTALL_PATH) || install -D -d -m 00755 $(BINARY_INSTALL_PATH)) && \
		install -m 00755 $(BINARY) $(BINARY_INSTALL_PATH)/ && \
		echo -e "\rInstalled at: $(BINARY_INSTALL)"; \
	);

.PHONY: clean
clean:
	@ ( \
		echo -en "Cleaning... "; \
		(test ! -d $(CUR_DIR)/pkg || rm -rf $(CUR_DIR)/pkg) && \
		(test ! -d $(CUR_DIR)/out || rm -rf $(CUR_DIR)/out) && \
		rm -rf $(BINARY)* && \
		echo -e "\rCleaned workspace."; \
	);

.PHONY: all
all: deps bin
