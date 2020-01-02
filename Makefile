.DEFAULT_GOAL := all
PROJECT_ROOT := 

NAME := spotitube
CUR_DIR = $(shell pwd)
BINARY_INSTALL_PATH := /usr/local/sbin
BINARY_INSTALL := $(BINARY_INSTALL_PATH)/$(NAME)
BINARY_PATH := $(CUR_DIR)/bin
BINARY := $(BINARY_PATH)/$(NAME)
VERSION := $(shell awk -F'= ' '/Version / {print $$2}' main.go | xargs)
PKG_NAME := $(BINARY)-v$(VERSION)
GOARCH := amd64
LDFLAGS := -s -w
OS := $(shell uname)

include Makefile.gobuild
include Makefile.pkg

.PHONY: install
install: bin
	@ ( \
		echo -en "Installing...\r"; \
		(test -d $(BINARY_INSTALL_PATH) || install -D -d -m 00755 $(BINARY_INSTALL_PATH)) && \
		install -m 00755 $(BINARY) $(BINARY_INSTALL_PATH)/ && \
		echo -e "\rInstalled at: $(BINARY_INSTALL)"; \
	);

.PHONY: clean
clean:
	@ ( \
		echo -en "Cleaning...\r"; \
		(test ! -d $(BINARY_PATH) || rm -rf $(BINARY_PATH)) && \
		rm -rf $(BINARY)* && \
		echo -e "\rCleaned workspace."; \
	);

.PHONY: all
all: deps bin
