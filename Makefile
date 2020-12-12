TARGET=net-hub
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --exact-match --tags $(git log -n1 --pretty='%h'))

VERSION_FLAGS=-ldflags '-s -w \
    -X github.com/JulienBalestra/net-hub/cmd/version.Version=$(VERSION) \
    -X github.com/JulienBalestra/net-hub/cmd/version.Commit=$(COMMIT)'

arm64:
	GOARCH=arm64 go build -i -o $(TARGET)-arm64 $(VERSION_FLAGS) .

amd64:
	go build -o $(TARGET)-amd64 $(VERSION_FLAGS) .

clean:
	$(RM) $(TARGET)-amd64 $(TARGET)-arm64

re: clean amd64 arm

fmt:
	@go fmt ./...

lint:
	golint -set_exit_status $(go list ./...)

import:
	goimports -w pkg/ cmd/ server/ client/

ineffassign:
	ineffassign ./

test:
	@go test -v -race ./...

vet:
	@go vet -v ./...

.pristine:
	git ls-files --exclude-standard --modified --deleted --others | diff /dev/null -

verify-fmt: fmt .pristine

verify-import: import .pristine
