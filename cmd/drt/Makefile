.PHONY: install clean

PROGRAM_NAME = $(notdir $(CURDIR))

INSTALL_CMD = go install -ldflags="-s -w"

BIN_PATH = $(shell go env GOPATH)/bin

install:
	@echo "Installing $(PROGRAM_NAME)$(shell go env GOEXE) to $(BIN_PATH)..."
	$(INSTALL_CMD)

clean:
	@echo "Removing $(PROGRAM_NAME)$(shell go env GOEXE)..."
	@-rm -f "$(BIN_PATH)/$(PROGRAM_NAME)$(shell go env GOEXE)" 2>/dev/null || true
	@-del /Q "$(subst /,\,$(BIN_PATH)/$(PROGRAM_NAME)$(shell go env GOEXE))" 2>nul || true

list:
	@echo "Project: $(PROGRAM_NAME)"
	@echo "OS: $(OS)"
ifeq ($(OS),Windows_NT)
	@dir "$(subst /,\,$(BIN_PATH)/$(PROGRAM_NAME)$(shell go env GOEXE))" 2>nul || echo "File not found"
else
	@ls -l "$(BIN_PATH)/$(PROGRAM_NAME)$(shell go env GOEXE)" 2>/dev/null || echo "File not found"
endif