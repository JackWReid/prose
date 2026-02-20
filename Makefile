# prose Makefile

# Installation directories
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1

# Binary name
BINARY := prose

# Read version from VERSION file
VERSION := $(shell cat VERSION)

# Build the prose binary
build:
	@echo "Building $(BINARY) v$(VERSION)..."
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/prose/
	@echo "Build complete: $(BINARY)"

# Install prose binary and man page
install: build
	@echo "Installing $(BINARY) to $(DESTDIR)$(BINDIR)..."
	mkdir -p $(DESTDIR)$(BINDIR)
	mkdir -p $(DESTDIR)$(MANDIR)
	install -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -m 0644 prose.1 $(DESTDIR)$(MANDIR)/prose.1
	@echo "Installation complete!"
	@echo "Binary: $(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "Man page: $(DESTDIR)$(MANDIR)/prose.1"

# Install just the man page (useful when binary is installed via go install)
install-man:
	@echo "Installing man page to $(DESTDIR)$(MANDIR)..."
	mkdir -p $(DESTDIR)$(MANDIR)
	install -m 0644 prose.1 $(DESTDIR)$(MANDIR)/prose.1
	@echo "Man page installed: $(DESTDIR)$(MANDIR)/prose.1"

# Uninstall prose binary and man page
uninstall:
	@echo "Uninstalling $(BINARY)..."
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -f $(DESTDIR)$(MANDIR)/prose.1
	@echo "Uninstall complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build and run
run: build
	./$(BINARY)

# Show help
help:
	@echo "prose Makefile targets:"
	@echo "  make build        - Build the prose binary"
	@echo "  make install      - Install prose binary and man page (default PREFIX=/usr/local)"
	@echo "  make install-man  - Install just the man page (for use with go install)"
	@echo "  make uninstall    - Remove installed prose binary and man page"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run all tests"
	@echo "  make run          - Build and run prose"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX            - Installation prefix (default: /usr/local)"
	@echo "  BINDIR            - Binary installation directory (default: PREFIX/bin)"
	@echo "  MANDIR            - Man page installation directory (default: PREFIX/share/man/man1)"
	@echo "  DESTDIR           - Staging directory for package builds"
	@echo ""
	@echo "Examples:"
	@echo "  make install PREFIX=/usr/local"
	@echo "  make install PREFIX=~/.local"
	@echo "  make install DESTDIR=/tmp/staging PREFIX=/usr"

.PHONY: build install install-man uninstall clean test run help
