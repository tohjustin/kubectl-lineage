# Stage 1: Build the Go binary
FROM golang:1.22 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kube-lineage ./cmd/kube-lineage

RUN ./kube-lineage --version
# Stage 2: Build the final image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/kube-lineage .

RUN ls -la

RUN ./kube-lineage --version

# Expose port 8080 to the outside world (optional, if your binary listens on a port)
EXPOSE 8080

# Command to run the executable
CMD ["./kube-lineage"]
