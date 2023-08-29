# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -p 8 -o rancher-tokens main.go

# Final stage
FROM alpine:latest

# Install ca-certificates, bash, curl, and libc6-compat
RUN apk add --no-cache ca-certificates bash curl sed libc6-compat && \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# Copy binary and bash scripts
COPY --from=builder /app/rancher-tokens /app/rancher-tokens

#COPY create-secret-on-cp-over-socks.sh generate-manifests.sh template-manifest-for-edge.yaml /app/
#COPY create-secret-on-cp-over-socks.sh /app/create-secret-on-cp-over-socks.sh
#COPY generate-manifests.sh /app/generate-manifests.sh
#COPY template-manifest-for-edge.yaml /app/template-manifest-for-edge.yaml

# Set the working directory
WORKDIR /app

# Make the bash scripts executable
#RUN chmod +x /app/create-secret-on-cp-over-socks.sh
#RUN chmod +x /app/generate-manifests.sh

# Expose the API port
#EXPOSE 8080

# Run the API
CMD ["/app/rancher-tokens"]
