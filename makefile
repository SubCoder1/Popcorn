# This MakeFile is going to build Popcorn.

# This is the filename of the executable to be created during build.
BINARY_NAME = Popcorn

# This line is to ensure the targets aren't considered as files.
.PHONY: load-dev-env dev-build dev test clean

# This target loads development environment variables via dev.conf.
load-dev-env:
	@echo "Loading dev.env"
    include config/dev.env
    export

# This target starts db in development environment.
load-db:
	sudo service redis-server restart

# [Unix] Build command for dev environment.
dev-build: load-dev-env
	@echo "Building Popcorn"
	go build -o ${BINARY_NAME}.exe ./cmd/server/.

# [Unix] Command to run the compiled file after build.
dev: dev-build
	./${BINARY_NAME}.exe

# Command to run all Popcorn tests.
test:
	go test -v ./...

# Clean up
clean:
	go clean
	rm ${BINARY_NAME}.exe