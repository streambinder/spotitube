.DEFAULT_GOAL := all
PROJECT_ROOT := src/


BINARIES = \
	spotify-dl

include Makefile.gobuild

_PKGS = \
	spotify-dl \
	spotify \
	utils \
	youtube

_DEPENDENCIES = \
	github.com/zmb3/spotify \
	github.com/mikkyang/id3-go \
	github.com/PuerkitoBio/goquery \
	github.com/fatih/color \
	github.com/kennygrant/sanitize

_CHECK_DEPENDENCIES = $(addsuffix .dependency,$(_DEPENDENCIES))

_CHECK_COMPLIANCE = $(addsuffix .compliant,$(_PKGS))

BINS = $(addsuffix .statbin,$(BINARIES))

dependencies: $(_CHECK_DEPENDENCIES)

compliant: $(_CHECK_COMPLIANCE)

install: $(BINS)
	test -d $(DESTDIR)/usr/local/bin || install -D -d -m 00755 $(DESTDIR)/usr/local/bin; \
	install -m 00755 bin/* $(DESTDIR)/usr/local/bin/; \

all: dependencies $(BINS)
