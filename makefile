# This MakeFile is going to build Popcorn.

# This is the filename of the executable to be created during build.
BINARY_NAME = Popcorn

# This line is to ensure the targets aren't considered as files.
.PHONY: dev-load-env dev-build dev clean

# This target loads development environment variables via dev.conf.
load-dev-env:
	@echo "Loading dev.env"
    include config/dev.env
    export

# This target loads test environment variables via test.conf.
load-test-env:
	@echo "Loading test.env"
    include config/test.env
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
test: load-test-env
	go test -v ./...

# Clean up
clean:
	go clean
	rm ${BINARY_NAME}.exe