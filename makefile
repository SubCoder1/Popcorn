# This MakeFile is going to build Popcorn.

# This is the filename of the executable to be created during build.
BINARY_NAME = Popcorn

# This line is to ensure the targets aren't considered as files.
.PHONY: dev-load-env dev-build dev clean

# This target loads development environment variables.
dev-load-env:
	@echo "Loading dev.env"
    include config/dev.env
    export

# [Unix] Build command for DEV Environment.
dev-build: dev-load-env
	@echo "Building Popcorn"
	go build -o ${BINARY_NAME}.exe ./cmd/server/.

# [Unix] Command to run the compiled file after build.
dev: dev-build
	./${BINARY_NAME}.exe

# Clean up
clean:
	go clean
	rm ${BINARY_NAME}.exe