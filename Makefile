# The three things you run by hand. CI runs the same ones.
#
#   make build   compile every tool into bin/, beside the swoop script
#   make test    gofmt, vet, and the unit tests
#   make bench   startup and list time of the hot-path tools (needs hyperfine)

.PHONY: build test bench clean

build:
	go build -o bin/ ./cmd/...

test:
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt: $$unformatted"; exit 1; fi
	go vet ./...
	go test ./...

bench: build
	hyperfine --warmup 5 -N 'bin/swoop-list'
	hyperfine --warmup 5 'bin/swoop-list | fzf --filter saf --delimiter "\t" --with-nth "{3} {4}" --nth 1'
	hyperfine --warmup 5 -N 'bin/swoop-preview /Applications/Safari.app'
	hyperfine --warmup 5 'bin/swoop-list | bin/swoop-icons -out /dev/null'
	SWOOP_EXTENSIONS=extensions hyperfine --warmup 5 -N 'extensions/define/define view define de'
	hyperfine --warmup 5 -N 'bin/swoop-dict swoop'

clean:
	find bin -type f ! -name swoop -delete
