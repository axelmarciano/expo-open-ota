DOCKER_FLAG := $(findstring docker, $(MAKECMDGOALS))
HTML_FLAG := $(findstring html, $(MAKECMDGOALS))
MAKEFLAGS += --silent

# Pinned so a new upstream release can't turn CI red without a code change.
DEADCODE_VERSION := v0.30.0
STATICCHECK_VERSION := v0.6.1

build:
ifeq ($(DOCKER_FLAG),docker)
	docker-compose build
else
	go build ./...
endif

up:
ifeq ($(DOCKER_FLAG),docker)
	docker-compose up -d
else
	reflex -r '\.go$$' -s -- sh -c "go run cmd/api/main.go"
endif

down:
ifeq ($(DOCKER_FLAG),docker)
	docker-compose down
else
	echo "Not applicable locally. Stop the application manually."
endif

# Both tools always run before failing, so one `make lint` reports every finding.
# They are complementary: staticcheck catches unused unexported identifiers (incl. struct
# fields), deadcode catches exported funcs that nothing reachable from main calls.
lint:
	rc=0; \
	echo "==> staticcheck U1000 (unused unexported identifiers)"; \
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) -checks 'U1000' ./... || rc=1; \
	echo "==> deadcode (funcs unreachable from main; -test keeps test-only helpers alive)"; \
	out=$$(go run golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION) -test ./...) || rc=1; \
	if [ -n "$$out" ]; then echo "$$out"; rc=1; fi; \
	if [ $$rc -ne 0 ]; then \
		echo "==> dead code found: delete it, or wire it up to a reachable path."; \
		exit 1; \
	fi; \
	echo "==> no dead code"

test_app:
ifeq ($(DOCKER_FLAG),docker)
	docker-compose --profile test run --rm -e "" ota-server-test go test -v -coverprofile=coverage.out ./...
else
	$(MAKE_COVERAGE_CMD)
endif

test_app_watch:
	find . -name '*.go' | entr -n -c $(MAKE) test_app $(DOCKER_FLAG) $(HTML_FLAG)


define MAKE_COVERAGE_CMD
	go test -v -coverprofile=coverage.out ./... && \
	$(call CLEAN_COVERAGE) && \
	$(call GENERATE_HTML)
endef

define CLEAN_COVERAGE
	if [ "$(shell uname -s)" = "Darwin" ]; then \
		sed -i '' -e '/test/d' -e '/cmd/d' coverage.out; \
	else \
		sed -i '/test/d;/cmd/d;' coverage.out; \
	fi
endef

define GENERATE_HTML
	if [ "$(HTML_FLAG)" = "html" ]; then \
		go tool cover -html=coverage.out -o coverage.html && \
		echo 'Coverage report generated: coverage.html'; \
	fi
endef

.PHONY: docker html lint
