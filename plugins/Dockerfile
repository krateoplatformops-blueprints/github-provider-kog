# Dockerfile in plugins/
# Optimized for granular Docker layer caching in a Go workspace monorepo.
# This Dockerfile ensures that changes in one plugin do not invalidate
# the build cache for other, unrelated plugins.

# --- Base Stage ---
# Defines the workspace directory and the PLUGIN_NAME build argument.
# Using a base stage keeps subsequent stages cleaner.
FROM golang:1.24-alpine AS base
WORKDIR /workspace
ARG PLUGIN_NAME

# --- Deps Stage ---
# This stage's sole purpose is to download Go module dependencies.
# It's structured to maximize layer caching.
FROM base AS deps

# 1. Copy the workspace definition. A change here invalidates all builds.
# The wildcard ensures this works even if go.work.sum doesn't exist yet.
COPY go.work* ./

# 2. Copy and download dependencies for the shared 'pkg' module first.
# This layer is cached as long as pkg's dependencies don't change.
COPY pkg/go.mod pkg/go.sum ./pkg/
RUN cd pkg && go mod download

# 3. Copy and download dependencies for ONLY the specific plugin being built.
# This is the key to build isolation. A change in another plugin's go.mod
# will not invalidate this layer for the current plugin's build.
COPY cmd/${PLUGIN_NAME}/go.mod cmd/${PLUGIN_NAME}/go.sum ./cmd/${PLUGIN_NAME}/
RUN cd cmd/${PLUGIN_NAME} && go mod download

# --- Builder Stage ---
# This stage compiles the specific plugin using the pre-downloaded dependencies.
FROM base AS builder

# 1. Copy the downloaded Go modules from the 'deps' stage.
COPY --from=deps /go/pkg/mod /go/pkg/mod

# 2. Copy the workspace definition again.
# The wildcard ensures this works even if go.work.sum doesn't exist yet.
COPY go.work* ./

# 3. Copy source code, being specific to minimize cache invalidation.
# Copy the shared 'pkg' source first, as it's a dependency for all plugins.
COPY pkg ./pkg
# Then, copy ONLY the source code for the specific plugin being built.
COPY cmd/${PLUGIN_NAME} ./cmd/${PLUGIN_NAME}

# 4. Build the target plugin.
# The command is run from the workspace root to ensure the go.work file is found and used.
RUN go build -o /app/server ./cmd/${PLUGIN_NAME}

# --- Final Stage ---
# Creates a minimal, non-root final image for the service.
FROM gcr.io/distroless/static-debian11

COPY --from=builder /app/server /server
USER nonroot:nonroot

ENTRYPOINT ["/server"]