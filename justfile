APP      := "gsqitch"
VERSION  := `perl -nE'm{version\s*=\s*"(\d+\.\d+.\d+)"} && print $1' ./cmd/sqitch/main.go`

build:
  echo "Building version {{VERSION}} of {{APP}}"
  go build -o gsqitch cmd/sqitch/main.go

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

release:
  git diff --exit-code
  git tag "{{VERSION}}"
  goreleaser release --clean
