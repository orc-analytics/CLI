.PHONY: all 

all: test
test: .test_all

# flags for stripping debugging info
LDFLAGS = -s -w

## CLI Build scripts
BINARY_NAME = orca

# disabled CGO to produce statically-linked binaries
export CGO_ENABLED = 0

.test_all:
	go test ./... -v

# ------------- BUILD -------------
#  build command
BUILD = go build -ldflags "$(LDFLAGS)" -o

# platform targets
.PHONY: all clean linux windows mac_arm mac_intel

build_all: linux windows mac_arm mac_intel

linux:
	cd GOOS=linux GOARCH=amd64 $(BUILD) ../build/$(BINARY_NAME)-amd64-cli-linux .

windows:
	cd GOOS=windows GOARCH=amd64 $(BUILD) ../build/$(BINARY_NAME)-amd64-cli-windows.exe .

mac_arm:
	cd GOOS=darwin GOARCH=arm64 $(BUILD) ../build/$(BINARY_NAME)-amd64-cli-mac-arm .

mac_intel:
	cd GOOS=darwin GOARCH=amd64 $(BUILD) ../build/$(BINARY_NAME)-amd64-cli-mac-intel .

clean:
	rm -rf build/
