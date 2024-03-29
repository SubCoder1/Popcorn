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

# Install cert package
RUN apk --no-cache add ca-certificates

# Setup container date/time
RUN apk add --no-cache tzdata

ENV TZ Asia/Kolkata

RUN cp /usr/share/zoneinfo/$TZ /etc/localtime

# Copy exec from builder
COPY --from=builder /Popcorn/main /usr/bin/server

EXPOSE 8080

# Run the executable.
CMD [ "server" ]