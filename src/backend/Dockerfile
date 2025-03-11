# syntax=docker/dockerfile:1.4
# escape=\

# Multi-stage Dockerfile for building and running the Document Management Platform services (API and worker)
# This Dockerfile follows best practices for creating minimal, secure container images suitable for deployment to Kubernetes.

# =====================================================================================================================
# Build Stage: Builder
# =====================================================================================================================
FROM golang:1.21-alpine AS builder
# golang:1.21-alpine - Base image for building the Go application

LABEL stage=builder

# Define build arguments
ARG SERVICE=api # Specifies which service to build (api or worker)
ARG VERSION=latest # Version tag for the image
ARG GIT_COMMIT=unknown # Git commit hash for traceability
ARG BUILD_DATE=unknown # Build timestamp for traceability

# Set environment variables
ENV APP_HOME=/app
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Install build dependencies
RUN apk add --no-cache git make bash

# Create app directory and set as working directory
RUN mkdir -p ${APP_HOME}
WORKDIR ${APP_HOME}

# Copy go.mod and go.sum files first for dependency caching
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy the rest of the source code
COPY . ./

# Build the application with version information embedded
RUN make build SERVICE=${SERVICE} VERSION=${VERSION} GIT_COMMIT=${GIT_COMMIT} BUILD_DATE=${BUILD_DATE}

# Create directories for data
RUN mkdir -p data

# =====================================================================================================================
# Runtime Stage: Final
# =====================================================================================================================
FROM alpine:3.17 AS final
# alpine:3.17 - Minimal base image for running the application

LABEL stage=final

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user and group
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -h /home/appuser -D appuser

# Create app directory with proper permissions
RUN mkdir -p /app && chown -R appuser:appuser /app

# Copy the compiled binary from the build stage
COPY --from=builder /app/bin/* /app/

# Set the working directory
WORKDIR /app

# Set environment variables for runtime
ENV SERVICE=${SERVICE}
ENV VERSION=${VERSION}
ENV GIT_COMMIT=${GIT_COMMIT}
ENV BUILD_DATE=${BUILD_DATE}

# Expose the application port
EXPOSE 8080/tcp # API service HTTP port

# Switch to non-root user
USER appuser

# Set the entrypoint to run the application
ENTRYPOINT ["/app/main", "-service=${SERVICE}"]