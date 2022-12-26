# This MakeFile is going to build Popcorn.

# This is the filename of the executable to be created during build.
BINARY_NAME = Popcorn

# This line is to ensure the targets aren't considered as files.
.PHONY: dev-load-env dev-build-win dev-win clean-win

# This target loads development environment variables.
dev-load-env:
	@echo "Loading dev.env"
    include .\config\dev.env
    export

# [Win] Build command for DEV Environment.
dev-build-win: dev-load-env
	@echo "Building Popcorn"
	go build -o ${BINARY_NAME}.exe .\cmd\server\.

# [Win] Command to run the compiled file after build.
dev-win: dev-build-win
	.\${BINARY_NAME}.exe

# Clean up
clean-win:
	go clean -modcache
	rm ${BINARY_NAME}.*