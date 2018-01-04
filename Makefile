ALL_SRC 	= $(shell find . -name "*.go" | grep -v -e vendor)
BINARY		=	$(shell echo $${PWD\#\#*/})
PACKAGES 	=	$(shell go list ./... | grep -v /vendor/)
RACE			=	-race
GOTEST		=	go test -v $(RACE)
GOLINT		=	golint
GOVET			=	go vet
GOFMT			=	gofmt
ERRCHECK	=	errcheck -ignoretests
FMT_LOG		=	fmt.log
LINT_LOG	=	lint.log
PASS			=	$(shell printf "\033[32mPASS\033[0m")
FAIL			=	$(shell printf "\033[31mFAIL\033[0m")
COLORIZE	=	sed ''/PASS/s//$(PASS)/'' | sed ''/FAIL/s//$(FAIL)/''

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(ALL_SRC) test fmt lint

.PHONY: install
install:
	dep version || (go get github.com/golang/dep/cmd/dep && dep version)
	dep ensure -v -vendor-only # assumes updated Gopkg.lock

.PHONY: fmt
fmt:
	@$(GOFMT) -e -s -l -w $(ALL_SRC)
	@./scripts/updateLicenses.sh

.PHONY: cover
cover:
	@./scripts/cover.sh $(shell go list $(PACKAGES))
	@go tool cover -html=cover.out -o cover.html

.PHONY: test
test:
	@bash -c "set -e; set -o pipefail; $(GOTEST) $(PACKAGES) | $(COLORIZE)"

.PHONY: lint
lint:
	@$(GOVET) $(PACKAGES)
	@$(ERRCHECK) $(PACKAGES)
	@cat /dev/null > $(LINT_LOG)
	@$(foreach pkg, $(PACKAGES), $(GOLINT) $(pkg) >> $(LINT_LOG) || true;)
	@[ ! -s "$(LINT_LOG)" ] || (echo "Lint Failures" | cat - $(LINT_LOG) && false)
	@$(GOFMT) -e -s -l $(ALL_SRC) > $(FMT_LOG)
	@[ ! -s "$(FMT_LOG)" ] || (echo "Go Fmt Failures, run 'make fmt'" | cat - $(FMT_LOG) && false)

.PHONY: install_ci
install_ci: install
	go get github.com/wadey/gocovmerge
	go get github.com/mattn/goveralls
	go get golang.org/x/tools/cmd/cover
	go get github.com/golang/lint/golint
	go get github.com/kisielk/errcheck