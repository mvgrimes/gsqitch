APP      := "gsqitch"
VER_FILE  := "./cmd/sqitch/main.go"
MAIN_FILE := "./cmd/sqitchxmux/main.go"
VERSION   := shell('perl -nE "m{version\\s*=\\s*\"(\\d+\\.\\d+\\.\\d+)\"}i && print \$1" ' + VER_FILE)

build:
  echo "Building version {{VERSION}} of {{APP}}"
  go build -o {{APP}} {{MAIN_FILE}}

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

fmt:
  go fmt ./...

test:
  just test-unit
  just test-integration

test-unit:
  # GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" go test ./...
  gotestsum

test-integration:
  # GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" go test -v -tags=integration ./internal/command
  GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" gotestsum -- -tags=integration ./internal/command

release:
  go mod tidy
  just fmt
  just build
  git diff --exit-code
  git tag --points-at HEAD | grep -qx {{VERSION}} || git tag {{VERSION}}
  git push
  git release
  git push --tags
  goreleaser release --clean
