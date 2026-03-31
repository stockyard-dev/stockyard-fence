.PHONY: build run test vet clean

build:
	CGO_ENABLED=0 go build -o fence ./cmd/fence/

run: build
	FENCE_ADMIN_KEY=dev-admin DATA_DIR=./data ./fence

test:
	go test ./... -count=1 -timeout 60s

vet:
	go vet ./...

clean:
	rm -f fence
