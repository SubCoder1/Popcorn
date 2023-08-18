FROM golang:1.21.0-alpine3.17 AS builder

# Set the Current Working Directory inside the container
WORKDIR /Popcorn
COPY . .

# Fetch and install dependencies.
RUN go mod download

# Build the binary.
RUN go build -o main ./cmd/server/.

# Production stage
FROM alpine:3.18.3 as production-stage

# Install ffmpeg
RUN apk update && apk add --no-cache ffmpeg

# Setup container date/time
RUN apk add --no-cache tzdata

ENV TZ Asia/Kolkata

RUN cp /usr/share/zoneinfo/$TZ /etc/localtime

# Copy exec from builder
COPY --from=builder /Popcorn/main /usr/bin/server

# Run the executable.
CMD [ "server" ]