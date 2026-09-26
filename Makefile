# The three things you run by hand. CI runs the same ones.
#
#   make build   compile every tool into bin/, beside the swoop script
#   make test    gofmt, vet, and the unit tests
#   make bench   startup and list time of the hot-path tools (needs hyperfine)
#   make e2e     the launcher driven through a pseudo-terminal (needs fzf)

.PHONY: build test bench e2e clean shell-mac install uninstall

build:
	go build -o bin/ ./cmd/...

# The macOS frame: a floating panel with swoop in it, no Ghostty.app needed.
# Run it with bin/ on PATH, or SWOOP_LAUNCHER pointing at bin/swoop.
shell-mac:
	swift build -c release --package-path shell/mac
	@echo "built shell/mac/.build/release/swoop-shell-mac"

# Daily driver: the frame and the clipboard watcher at login, through launchd.
install:
	scripts/install

uninstall:
	scripts/uninstall

test:
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt: $$unformatted"; exit 1; fi
	go vet ./...
	go test ./...
	scripts/launchd-test

# The launcher end to end: bin/swoop driven through a pseudo-terminal with
# a fake extension and a fake AI command. Needs fzf and python3.
e2e: build
	scripts/launcher-test

bench: build
	hyperfine --warmup 5 -N 'bin/swoop-list'
	hyperfine --warmup 5 'bin/swoop-list | fzf --filter saf --delimiter "\t" --with-nth "{3} {4}" --nth 1'
	hyperfine --warmup 5 -N 'bin/swoop-preview /Applications/Safari.app'
	hyperfine --warmup 5 'bin/swoop-list | bin/swoop-icons -out /dev/null'
	SWOOP_EXTENSIONS=extensions hyperfine --warmup 5 -N 'extensions/define/define view define de'
	hyperfine --warmup 5 -N 'bin/swoop-dict swoop'
	SWOOP_EXTENSIONS=extensions hyperfine --warmup 5 -N 'bin/swoop-nav rows 2+2'
	hyperfine --warmup 5 -N 'bin/swoop-clipd list'

clean:
	find bin -type f ! -name swoop -delete
