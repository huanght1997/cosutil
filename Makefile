export PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on
LDFLAGS := -s -w

all: build

build: app

app:
	env CGO_ENABLED=0 GOOS=darwin GOARCH=386 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_darwin_386
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_darwin_amd64
	env CGO_ENABLED=0 GOOS=freebsd GOARCH=386 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_freebsd_386
	env CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_freebsd_amd64
	env CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_linux_386
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_linux_amd64
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_linux_arm
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_linux_arm64
	env CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_windows_386.exe
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ./release/cosutil_windows_amd64.exe

clean:
	rm -rf ./release
