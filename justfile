fmt:
	gofumpt -w .

lint:
	golangci-lint run

test:
	go test ./... -cover

check:
	go vet ./...

build:
	go build ./...

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

snap:
	node scripts/ghostty-web-snap.mjs

termcaps-e2e:
	bash scripts/termcaps-e2e-macos.sh

termcaps-e2e-gui:
	bash scripts/termcaps-e2e-gui-macos.sh
