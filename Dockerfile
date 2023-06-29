FROM golang:alpine AS builder

# Set the Current Working Directory inside the container
RUN mkdir /Popcorn
WORKDIR /Popcorn
ADD . /Popcorn

# Fetch dependencies.
# Using go get.
RUN go mod download

# Build the binary.
RUN go build -o main ./cmd/server/.

FROM alpine
RUN apk update && apk upgrade && apk add --no-cache 'git=~2' && apk add --no-cache ffmpeg
COPY --from=builder /Popcorn/main /usr/bin/server 

# Run the executable.
CMD [ "server" ]