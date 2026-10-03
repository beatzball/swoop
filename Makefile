# The three things you run by hand. CI runs the same ones.
#
#   make build   compile every tool into bin/, beside the swoop script
#   make test    build, then gofmt, vet, the unit tests, and the script tests
#   make bench   every hot path as one line of JSON (needs hyperfine and fzf)
#   make e2e     the launcher driven through a pseudo-terminal (needs fzf)

.PHONY: build test bench e2e clean shell-mac install uninstall

build:
	go build -o bin/ ./cmd/...

# The macOS frame: a floating panel with the launcher in it, no Ghostty.app
# needed. Run it with bin/ on PATH, or SWOOP_LAUNCHER pointing at bin/swoop.
# Signed with <name>-dev, the tool file's name, when this Mac has it, so an
# Accessibility grant survives the rebuild; see scripts/sign-frame.
shell-mac:
	swift build -c release --package-path shell/mac
	@scripts/sign-frame shell/mac/.build/release/swoop-shell-mac
	@echo "built shell/mac/.build/release/swoop-shell-mac"

# Daily driver: the frame and the clipboard watcher at login, through launchd.
install:
	scripts/install

uninstall:
	scripts/uninstall

# Builds first: scripts/template-test runs swoop-check from bin/, and a
# fresh worktree has only the swoop script there. CI spells its steps out
# and builds once on its own, so this costs it nothing.
test: build
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt: $$unformatted"; exit 1; fi
	go vet ./...
	go test ./...
	scripts/launchd-test
	scripts/template-test
	scripts/reminders-test
	scripts/codesign-test
	scripts/contract-test

# The launcher end to end: bin/swoop driven through a pseudo-terminal with
# a fake extension and a fake AI command. Needs fzf and python3.
e2e: build
	scripts/launcher-test

# Every hot path as one line of JSON: the tools one process each, the keys
# through the launcher itself, the frame and the sizes. The budgets are at
# the top of the script. CI runs the same on every pull request.
bench: build
	@scripts/bench

clean:
	find bin -type f ! -name swoop -delete
