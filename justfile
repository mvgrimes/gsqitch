APP      := "gsqitch"
VERSION  := `perl -nE'm{version\s*=\s*"(\d+\.\d+.\d+)"} && print $1' ./cmd/sqitch/main.go`

build:
  echo "Building version {{VERSION}} of {{APP}}"
  go build -o gsqitch cmd/sqitch/main.go

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

fmt:
  go fmt ./...

test:
  # GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" go test ./...
  gotestsum

test-integration:
  GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" go test -v -tags=integration ./internal/command
  # GSQITCH_TEST_TARGET="db:mysql://sqitch:sqitch@localhost:3307/sqitch" gotestsum -- -tags=integration ./internal/command

release:
  go mod tidy
  just fmt
  just build
  git diff --exit-code
  git tag "{{VERSION}}"
  git push
  git release
  git push --tags
  goreleaser release --clean
