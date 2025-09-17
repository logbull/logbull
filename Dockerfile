# ========= BUILD FRONTEND =========
FROM node:24-alpine AS frontend-build

WORKDIR /frontend

# Add version for the frontend build
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./

# Copy .env.production.example to .env
RUN if [ -f .env.production.example ]; then \
    cp .env.production.example .env; \
    else \
    echo "Error: .env.production.example not found" && exit 1; \
    fi

RUN npm run build

# ========= BUILD BACKEND =========
# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.23.3 AS backend-build

# Make TARGET args available early so tools built here match the final image arch
ARG TARGETOS
ARG TARGETARCH

# Install goose and swag tools
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Set working directory
WORKDIR /app

# Install Go dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Create required directories for embedding
RUN mkdir -p /app/ui/build

# Copy frontend build output for embedding
COPY --from=frontend-build /frontend/dist /app/ui/build

# Generate Swagger documentation
COPY backend/ ./
RUN swag init -d . -g cmd/main.go -o swagger

# Compile the backend
ARG TARGETVARIANT
RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -o /app/main ./cmd/main.go


# ========= RUNTIME =========
FROM debian:bookworm-slim

# Add version metadata to runtime image
ARG APP_VERSION=dev
LABEL org.opencontainers.image.version=$APP_VERSION
ENV APP_VERSION=$APP_VERSION

# Set production mode for Docker containers
ENV ENV_MODE=production

# Install PostgreSQL server, OpenSearch, Valkey, and Java
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget ca-certificates gnupg lsb-release sudo gosu curl openjdk-17-jre-headless build-essential && \
    wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/pgdg.gpg && \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
    > /etc/apt/sources.list.d/pgdg.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    postgresql-17 postgresql-client-17 && \
    rm -rf /var/lib/apt/lists/*

# Install OpenSearch
RUN wget https://artifacts.opensearch.org/releases/bundle/opensearch/2.12.0/opensearch-2.12.0-linux-x64.tar.gz && \
    tar -xzf opensearch-2.12.0-linux-x64.tar.gz && \
    mv opensearch-2.12.0 /opt/opensearch && \
    rm opensearch-2.12.0-linux-x64.tar.gz && \
    groupadd -g 1000 opensearch && \
    useradd -u 1000 -g opensearch -d /opt/opensearch -s /bin/bash opensearch && \
    mkdir -p /logbull-data/opensearch-data /logbull-data/opensearch-logs && \
    chown -R opensearch:opensearch /opt/opensearch /logbull-data/opensearch-data /logbull-data/opensearch-logs

# Install Valkey
RUN wget https://github.com/valkey-io/valkey/archive/refs/tags/8.0.1.tar.gz -O valkey-8.0.1.tar.gz && \
    tar -xzf valkey-8.0.1.tar.gz && \
    cd valkey-8.0.1 && \
    make && \
    make install PREFIX=/opt/valkey && \
    cd .. && \
    rm -rf valkey-8.0.1 valkey-8.0.1.tar.gz && \
    groupadd -g 1001 valkey && \
    useradd -u 1001 -g valkey -d /opt/valkey -s /bin/bash valkey && \
    mkdir -p /logbull-data/valkey-data /opt/valkey/conf && \
    chown -R valkey:valkey /opt/valkey /logbull-data/valkey-data

# Create postgres user and set up directories
RUN useradd -m -s /bin/bash postgres || true && \
    mkdir -p /logbull-data/pgdata && \
    chown -R postgres:postgres /logbull-data/pgdata

# Configure Valkey for high-performance caching
RUN echo '# Valkey Configuration for LogBull' > /opt/valkey/conf/valkey.conf && \
    echo 'bind 127.0.0.1' >> /opt/valkey/conf/valkey.conf && \
    echo 'port 6379' >> /opt/valkey/conf/valkey.conf && \
    echo 'timeout 0' >> /opt/valkey/conf/valkey.conf && \
    echo 'tcp-keepalive 300' >> /opt/valkey/conf/valkey.conf && \
    echo 'daemonize no' >> /opt/valkey/conf/valkey.conf && \
    echo 'loglevel notice' >> /opt/valkey/conf/valkey.conf && \
    echo 'logfile ""' >> /opt/valkey/conf/valkey.conf && \
    echo 'databases 16' >> /opt/valkey/conf/valkey.conf && \
    echo 'dir /logbull-data/valkey-data' >> /opt/valkey/conf/valkey.conf && \
    echo 'save 900 1' >> /opt/valkey/conf/valkey.conf && \
    echo 'save 300 10' >> /opt/valkey/conf/valkey.conf && \
    echo 'save 60 10000' >> /opt/valkey/conf/valkey.conf && \
    echo 'stop-writes-on-bgsave-error yes' >> /opt/valkey/conf/valkey.conf && \
    echo 'rdbcompression yes' >> /opt/valkey/conf/valkey.conf && \
    echo 'rdbchecksum yes' >> /opt/valkey/conf/valkey.conf && \
    echo 'dbfilename dump.rdb' >> /opt/valkey/conf/valkey.conf && \
    echo 'maxmemory-policy allkeys-lru' >> /opt/valkey/conf/valkey.conf && \
    echo 'appendonly no' >> /opt/valkey/conf/valkey.conf && \
    echo '# No password required as requested' >> /opt/valkey/conf/valkey.conf && \
    chown valkey:valkey /opt/valkey/conf/valkey.conf

# Configure OpenSearch for high-performance log storage
RUN echo '# Cluster configuration' > /opt/opensearch/config/opensearch.yml && \
    echo 'cluster.name: logbull-cluster' >> /opt/opensearch/config/opensearch.yml && \
    echo 'node.name: logbull-node' >> /opt/opensearch/config/opensearch.yml && \
    echo 'discovery.type: single-node' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Path configuration' >> /opt/opensearch/config/opensearch.yml && \
    echo 'path.data: /logbull-data/opensearch-data' >> /opt/opensearch/config/opensearch.yml && \
    echo 'path.logs: /logbull-data/opensearch-logs' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Network configuration' >> /opt/opensearch/config/opensearch.yml && \
    echo 'network.host: 127.0.0.1' >> /opt/opensearch/config/opensearch.yml && \
    echo 'http.port: 9200' >> /opt/opensearch/config/opensearch.yml && \
    echo 'transport.port: 9300' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Security configuration' >> /opt/opensearch/config/opensearch.yml && \
    echo 'plugins.security.disabled: true' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Memory and performance configuration' >> /opt/opensearch/config/opensearch.yml && \
    echo 'bootstrap.memory_lock: false' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.memory.index_buffer_size: 30%' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.memory.min_index_buffer_size: 96mb' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Index configuration for log storage' >> /opt/opensearch/config/opensearch.yml && \
    echo 'action.auto_create_index: true' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.query.bool.max_clause_count: 10000' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.fielddata.cache.size: 40%' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Thread pool configuration' >> /opt/opensearch/config/opensearch.yml && \
    echo 'thread_pool.write.queue_size: 1000' >> /opt/opensearch/config/opensearch.yml && \
    echo 'thread_pool.search.queue_size: 1000' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Circuit breaker settings' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.breaker.total.limit: 80%' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.breaker.fielddata.limit: 50%' >> /opt/opensearch/config/opensearch.yml && \
    echo 'indices.breaker.request.limit: 40%' >> /opt/opensearch/config/opensearch.yml && \
    echo '' >> /opt/opensearch/config/opensearch.yml && \
    echo '# Additional performance optimizations for log storage' >> /opt/opensearch/config/opensearch.yml && \
    echo 'cluster.routing.allocation.disk.threshold_enabled: true' >> /opt/opensearch/config/opensearch.yml && \
    echo 'cluster.routing.allocation.disk.watermark.low: 85%' >> /opt/opensearch/config/opensearch.yml && \
    echo 'cluster.routing.allocation.disk.watermark.high: 90%' >> /opt/opensearch/config/opensearch.yml && \
    echo 'cluster.routing.allocation.disk.watermark.flood_stage: 95%' >> /opt/opensearch/config/opensearch.yml

WORKDIR /app

# Copy Goose from build stage (installed via go install to /go/bin)
COPY --from=backend-build /go/bin/goose /usr/local/bin/goose

# Copy app binary 
COPY --from=backend-build /app/main .

# Copy go.mod for BackendRootPath detection
COPY backend/go.mod ./go.mod

# Copy migrations directory
COPY backend/migrations ./migrations

# Copy UI files
COPY --from=backend-build /app/ui/build ./ui/build

# Copy .env file (with fallback to .env.production.example)
COPY backend/.env* /app/
RUN if [ ! -f /app/.env ]; then \
    if [ -f /app/.env.production.example ]; then \
    cp /app/.env.production.example /app/.env; \
    fi; \
    fi

# Create startup script
COPY <<EOF /app/start.sh
#!/bin/bash
set -e

# PostgreSQL 17 binary paths
PG_BIN="/usr/lib/postgresql/17/bin"

# Ensure proper ownership of data directories
echo "Setting up data directory permissions..."
mkdir -p /logbull-data/pgdata /logbull-data/opensearch-data /logbull-data/opensearch-logs /logbull-data/valkey-data
chown -R postgres:postgres /logbull-data/pgdata
chown -R opensearch:opensearch /logbull-data/opensearch-data /logbull-data/opensearch-logs
chown -R valkey:valkey /logbull-data/valkey-data

# Initialize PostgreSQL if not already initialized
if [ ! -s "/logbull-data/pgdata/PG_VERSION" ]; then
    echo "Initializing PostgreSQL database..."
    gosu postgres \$PG_BIN/initdb -D /logbull-data/pgdata --encoding=UTF8 --locale=C.UTF-8
    
    # Configure PostgreSQL - basic settings (resource scaling done later)
    echo "host all all 127.0.0.1/32 md5" >> /logbull-data/pgdata/pg_hba.conf
    echo "local all all trust" >> /logbull-data/pgdata/pg_hba.conf
    echo "port = 5437" >> /logbull-data/pgdata/postgresql.conf
    echo "listen_addresses = 'localhost'" >> /logbull-data/pgdata/postgresql.conf
fi

# Start PostgreSQL in background
echo "Starting PostgreSQL..."
gosu postgres \$PG_BIN/postgres -D /logbull-data/pgdata -p 5437 &
POSTGRES_PID=\$!

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if gosu postgres \$PG_BIN/pg_isready -p 5437 -h localhost >/dev/null 2>&1; then
        echo "PostgreSQL is ready!"
        break
    fi
    if [ \$i -eq 30 ]; then
        echo "PostgreSQL failed to start"
        exit 1
    fi
    sleep 1
done

# Calculate and configure PostgreSQL resources (25% RAM, 100-300 connections)
echo "Configuring PostgreSQL resources..."
CPU_COUNT=\$(nproc)
PG_MEMORY_MB=\$((TOTAL_MEMORY_MB / 4))  # 25% of total memory

# PostgreSQL memory limits: minimum 256MB, maximum 25% of total RAM
if [ \$PG_MEMORY_MB -lt 256 ]; then
    PG_MEMORY_MB=256
fi

# PostgreSQL shared_buffers (typically 25% of allocated PostgreSQL memory)
PG_SHARED_BUFFERS_MB=\$((PG_MEMORY_MB / 4))
if [ \$PG_SHARED_BUFFERS_MB -lt 64 ]; then
    PG_SHARED_BUFFERS_MB=64
fi

# PostgreSQL connections scaling with CPU count (100-300 range)
PG_MAX_CONNECTIONS=\$((100 + (CPU_COUNT * 2)))
if [ \$PG_MAX_CONNECTIONS -gt 300 ]; then
    PG_MAX_CONNECTIONS=300
fi

echo "PostgreSQL memory allocation: \${PG_MEMORY_MB}MB (shared_buffers: \${PG_SHARED_BUFFERS_MB}MB)"
echo "PostgreSQL max connections: \${PG_MAX_CONNECTIONS} (based on \${CPU_COUNT} CPUs)"

# Apply PostgreSQL configuration
echo "shared_buffers = \${PG_SHARED_BUFFERS_MB}MB" >> /logbull-data/pgdata/postgresql.conf
echo "max_connections = \${PG_MAX_CONNECTIONS}" >> /logbull-data/pgdata/postgresql.conf
echo "effective_cache_size = \${PG_MEMORY_MB}MB" >> /logbull-data/pgdata/postgresql.conf
echo "maintenance_work_mem = \$((PG_MEMORY_MB / 16))MB" >> /logbull-data/pgdata/postgresql.conf
echo "checkpoint_completion_target = 0.9" >> /logbull-data/pgdata/postgresql.conf
echo "wal_buffers = 16MB" >> /logbull-data/pgdata/postgresql.conf
echo "default_statistics_target = 100" >> /logbull-data/pgdata/postgresql.conf
echo "random_page_cost = 1.1" >> /logbull-data/pgdata/postgresql.conf
echo "effective_io_concurrency = \$CPU_COUNT" >> /logbull-data/pgdata/postgresql.conf

# Create database and set password for postgres user
echo "Setting up database and user..."
gosu postgres \$PG_BIN/psql -p 5437 -h localhost -d postgres << 'SQL'
ALTER USER postgres WITH PASSWORD 'Q1234567';
SELECT 'CREATE DATABASE logbull OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'logbull')
\\gexec
\\q
SQL

# Start Valkey in background
echo "Starting Valkey..."
gosu valkey /opt/valkey/bin/valkey-server /opt/valkey/conf/valkey.conf &
VALKEY_PID=\$!

# Wait for Valkey to be ready
echo "Waiting for Valkey to be ready..."
for i in {1..30}; do
    if /opt/valkey/bin/valkey-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
        echo "Valkey is ready!"
        break
    fi
    if [ \$i -eq 30 ]; then
        echo "Valkey failed to start"
        exit 1
    fi
    sleep 1
done

# Calculate optimal memory allocation for OpenSearch (50% of available RAM)
echo "Calculating optimal memory allocation for OpenSearch..."
TOTAL_MEMORY_KB=\$(grep MemTotal /proc/meminfo | awk '{print \$2}')
TOTAL_MEMORY_MB=\$((TOTAL_MEMORY_KB / 1024))
OPENSEARCH_HEAP_MB=\$((TOTAL_MEMORY_MB / 2))

# Ensure minimum 512MB and maximum 31GB (OpenSearch recommendation)
if [ \$OPENSEARCH_HEAP_MB -lt 512 ]; then
    OPENSEARCH_HEAP_MB=512
elif [ \$OPENSEARCH_HEAP_MB -gt 31744 ]; then
    OPENSEARCH_HEAP_MB=31744
fi

echo "=== Resource Allocation Summary ==="
echo "Total system memory: \${TOTAL_MEMORY_MB}MB, CPUs: \${CPU_COUNT}"
echo "OpenSearch heap: \${OPENSEARCH_HEAP_MB}MB (50% of RAM, max 31GB)"
echo "PostgreSQL memory: \${PG_MEMORY_MB}MB (25% of RAM)"
echo "App + Valkey memory: \$((TOTAL_MEMORY_MB - OPENSEARCH_HEAP_MB - PG_MEMORY_MB))MB (remaining 25%)"
echo "==================================="

# Configure JVM options dynamically
export OPENSEARCH_JAVA_OPTS="-Xms\${OPENSEARCH_HEAP_MB}m -Xmx\${OPENSEARCH_HEAP_MB}m -XX:+UseG1GC -XX:G1HeapRegionSize=16m -XX:+UseLargePages -XX:+UnlockExperimentalVMOptions -XX:+UseTransparentHugePages -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Djava.io.tmpdir=/logbull-data/opensearch-data/tmp"

# Create temp directory for OpenSearch
mkdir -p /logbull-data/opensearch-data/tmp
chown opensearch:opensearch /logbull-data/opensearch-data/tmp

# Start OpenSearch in background
echo "Starting OpenSearch..."
gosu opensearch /opt/opensearch/bin/opensearch &
OPENSEARCH_PID=\$!

# Wait for OpenSearch to be ready
echo "Waiting for OpenSearch to be ready..."
for i in {1..120}; do
    if curl -s "http://127.0.0.1:9200/_cluster/health" >/dev/null 2>&1; then
        echo "OpenSearch is ready!"
        break
    fi
    if [ \$i -eq 120 ]; then
        echo "OpenSearch failed to start"
        exit 1
    fi
    echo "Waiting for OpenSearch... (\$i/120)"
    sleep 2
done

# Start the main application
echo "Starting Log Bull application..."
exec ./main
EOF

RUN chmod +x /app/start.sh

EXPOSE 4005

# Volume for PostgreSQL and OpenSearch data
VOLUME ["/logbull-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []