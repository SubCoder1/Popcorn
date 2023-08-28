# This MakeFile is going to build Popcorn.

# This is the filename of the executable to be created during build.
BINARY_NAME = Popcorn

# This line is to ensure the targets aren't considered as files.
.PHONY: load-local-env local-build local test clean

# This target loads development environment variables via dev.conf.
load-local-env:
	@echo "Loading local.env"
    include config/local.env
    include config/secrets.env
    export

# This target starts db in development environment.
load-db:
	sudo service redis-server restart

# [Unix] Build command for dev environment.
local-build: load-local-env
	@echo "Building Popcorn"
	go build -o ${BINARY_NAME}.exe ./cmd/server/.

# [Unix] Command to run the compiled file after build.
local: local-build
	./${BINARY_NAME}.exe

# Command to run all Popcorn tests.
test:
	go test -v ./...

# Clean up
clean:
	go clean
	rm ${BINARY_NAME}.exe