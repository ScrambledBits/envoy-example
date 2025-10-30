# Envoy Proxy Demo with Docker Compose

This demo showcases Envoy as an edge proxy managing traffic to multiple Go web service instances. It demonstrates load balancing, health checking, circuit breaking, and retry policies in a microservices architecture.

## Features

- **Load Balancing**: Round-robin distribution across 3 service instances
- **Health Checks**: Automatic detection and removal of unhealthy instances
- **Circuit Breaking**: Protection against cascading failures
- **Retry Logic**: Automatic retry on failures with smart host selection
- **Graceful Shutdown**: Zero-downtime deployments
- **Resource Management**: CPU and memory limits for all services
- **Structured Logging**: Comprehensive request logging
- **Optimized Images**: Multi-stage builds reducing image size by 95%

## Architecture

### System Overview

```mermaid
graph TB
    Client[Client/Browser]
    Envoy[Envoy Proxy<br/>:10000]
    Admin[Admin Interface<br/>:9901]
    WS1[Web Service 1<br/>:1337]
    WS2[Web Service 2<br/>:1337]
    WS3[Web Service 3<br/>:1337]

    Client -->|HTTP Request| Envoy
    Envoy -->|Round Robin| WS1
    Envoy -->|Load Balance| WS2
    Envoy -->|Health Check| WS3
    Envoy -.->|Metrics/Stats| Admin

    style Envoy fill:#f9f,stroke:#333,stroke-width:4px
    style Client fill:#bbf,stroke:#333,stroke-width:2px
    style Admin fill:#bfb,stroke:#333,stroke-width:2px
    style WS1 fill:#fbb,stroke:#333,stroke-width:2px
    style WS2 fill:#fbb,stroke:#333,stroke-width:2px
    style WS3 fill:#fbb,stroke:#333,stroke-width:2px
```

### Request Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant E as Envoy Proxy
    participant LB as Load Balancer
    participant W1 as Web Service 1
    participant W2 as Web Service 2
    participant W3 as Web Service 3

    C->>E: GET / HTTP/1.1
    Note over E: Check Circuit Breaker
    E->>LB: Route Request
    Note over LB: Round Robin Selection
    LB->>W2: Forward Request
    W2->>W2: Process Request
    W2-->>LB: 200 OK + Response
    LB-->>E: Response
    Note over E: Log Request Time
    E-->>C: 200 OK + Response

    Note over C,W3: If W2 fails, retry with another host
    C->>E: GET / HTTP/1.1
    E->>LB: Route Request
    LB->>W2: Forward Request
    W2--xLB: Connection Refused
    Note over E: Retry Policy Activated
    E->>LB: Retry Different Host
    LB->>W3: Forward Request
    W3-->>LB: 200 OK + Response
    LB-->>E: Response
    E-->>C: 200 OK + Response
```

### Health Check Flow

```mermaid
sequenceDiagram
    participant E as Envoy Proxy
    participant W1 as Web Service 1
    participant W2 as Web Service 2
    participant W3 as Web Service 3

    loop Every 10 seconds
        E->>W1: GET /health
        W1-->>E: 200 OK
        Note over W1: Healthy ✓

        E->>W2: GET /health
        W2--xE: Timeout
        Note over W2: Unhealthy (1/3)

        E->>W3: GET /health
        W3-->>E: 200 OK
        Note over W3: Healthy ✓
    end

    loop 3 Failed Checks
        E->>W2: GET /health
        W2--xE: Timeout
        Note over W2: Unhealthy (3/3)
    end

    Note over E,W2: W2 Removed from Load Balancer

    rect rgb(255, 200, 200)
        Note over W2: Service Ejected<br/>No Traffic Routed
    end

    loop Health Recovery
        E->>W2: GET /health
        W2-->>E: 200 OK
        Note over W2: Healthy (1/2)

        E->>W2: GET /health
        W2-->>E: 200 OK
        Note over W2: Healthy (2/2)
    end

    rect rgb(200, 255, 200)
        Note over W2: Service Restored<br/>Traffic Resumed
    end
```

### Circuit Breaker States

```mermaid
stateDiagram-v2
    [*] --> Closed
    Closed --> Open: 5 consecutive 5xx errors
    Open --> HalfOpen: After 30s (base ejection time)
    HalfOpen --> Closed: 2 successful requests
    HalfOpen --> Open: Any failure
    Open --> [*]: Max ejection time reached

    note right of Closed
        Normal Operation
        - All requests pass through
        - Tracking error rate
    end note

    note right of Open
        Circuit Breaker Active
        - Host ejected from pool
        - No traffic routed
        - Waiting for recovery
    end note

    note right of HalfOpen
        Testing Recovery
        - Limited requests allowed
        - Checking if service recovered
    end note
```

### Docker Compose Deployment

```mermaid
graph TB
    subgraph "Docker Network: proxy"
        subgraph "Envoy Container"
            EP[Envoy Process]
            EC[envoy.yaml config]
            EP -.->|reads| EC
        end

        subgraph "Web Service Containers"
            WS1[web-service-1<br/>Go Binary]
            WS2[web-service-2<br/>Go Binary]
            WS3[web-service-3<br/>Go Binary]
        end

        subgraph "Health Checks"
            HC1[Health Check<br/>every 10s]
            HC2[Health Check<br/>every 10s]
            HC3[Health Check<br/>every 10s]
        end

        HC1 -.->|monitors| WS1
        HC2 -.->|monitors| WS2
        HC3 -.->|monitors| WS3

        EP -->|load balance| WS1
        EP -->|load balance| WS2
        EP -->|load balance| WS3
    end

    subgraph "Host Machine"
        P1[":10000<br/>Main Traffic"]
        P2[":9901<br/>Admin UI"]
    end

    P1 -->|exposed| EP
    P2 -->|exposed| EP

    subgraph "Resource Limits"
        R1["WS1: 0.5 CPU, 128MB RAM"]
        R2["WS2: 0.5 CPU, 128MB RAM"]
        R3["WS3: 0.5 CPU, 128MB RAM"]
        R4["Envoy: 1.0 CPU, 256MB RAM"]
    end

    R1 -.->|limits| WS1
    R2 -.->|limits| WS2
    R3 -.->|limits| WS3
    R4 -.->|limits| EP

    style EP fill:#f9f,stroke:#333,stroke-width:4px
    style WS1 fill:#fbb,stroke:#333,stroke-width:2px
    style WS2 fill:#fbb,stroke:#333,stroke-width:2px
    style WS3 fill:#fbb,stroke:#333,stroke-width:2px
```

## Prerequisites

- Docker and Docker Compose installed on your local machine

## Quick Start

### 1. Start All Services

```bash
docker-compose up --build
```

This will:
- Build optimized Docker images for the Go web services
- Pull and configure the latest Envoy proxy
- Start 3 web service instances with health checks
- Start Envoy with load balancing configured

### 2. Test the Setup

**Send requests through the proxy:**
```bash
curl http://localhost:10000
```

You'll see responses from different service instances:
```
Hello, World! From web-service-1
Hello, World! From web-service-2
Hello, World! From web-service-3
```

**Check health endpoint:**
```bash
curl http://localhost:10000/health
```

**View Envoy admin interface:**
```bash
curl http://localhost:9901/stats
curl http://localhost:9901/clusters
```

### 3. Stop Services

```bash
docker-compose down
```

## Project Structure

```mermaid
graph TD
    Root[envoy-example/]

    Root --> DC[docker-compose.yml]
    Root --> MK[Makefile]
    Root --> RM[README.md]
    Root --> GWS[go-web-service/]
    Root --> EP[envoy-proxy/]

    GWS --> Main[main.go<br/>149 lines]
    GWS --> DF1[Dockerfile<br/>Multi-stage]
    GWS --> GM[go.mod]
    GWS --> GS[go.sum]
    GWS --> DI[.dockerignore]

    EP --> EY[envoy.yaml<br/>100 lines]
    EP --> DF2[Dockerfile<br/>Envoy v1.31]

    DC -.->|defines| SVC1[web-service-1]
    DC -.->|defines| SVC2[web-service-2]
    DC -.->|defines| SVC3[web-service-3]
    DC -.->|defines| SVCE[envoy-proxy]

    Main -.->|implements| CFG[Config Struct]
    Main -.->|implements| LOG[Logger Struct]
    Main -.->|implements| HND[HTTP Handlers]
    Main -.->|implements| GS2[Graceful Shutdown]

    EY -.->|configures| HC[Health Checks]
    EY -.->|configures| CB[Circuit Breakers]
    EY -.->|configures| RT[Retry Policy]
    EY -.->|configures| LB[Load Balancing]

    style Root fill:#f9f,stroke:#333,stroke-width:4px
    style Main fill:#bfb,stroke:#333,stroke-width:2px
    style EY fill:#bbf,stroke:#333,stroke-width:2px
    style DC fill:#fbb,stroke:#333,stroke-width:2px
    style MK fill:#ffb,stroke:#333,stroke-width:2px
```

**Directory Contents:**
```
.
├── docker-compose.yml          # Orchestrates all services
├── Makefile                    # Development commands
├── README.md                   # This file
├── go-web-service/
│   ├── main.go                 # Go service with graceful shutdown
│   ├── Dockerfile              # Multi-stage optimized build
│   ├── go.mod                  # Go module definition
│   ├── go.sum                  # Dependency checksums (if any)
│   └── .dockerignore           # Build optimization
└── envoy-proxy/
    ├── envoy.yaml              # Envoy configuration
    └── Dockerfile              # Envoy container setup
```

## Configuration

### Go Web Service

The service can be configured via environment variables:
- `PORT`: Server port (default: 1337)

See `go-web-service/main.go` for the complete implementation.

### Envoy Proxy

Key features configured in `envoy-proxy/envoy.yaml`:
- **Timeouts**: 30s overall, 10s per attempt
- **Retries**: Up to 3 retries on 5xx errors and connection failures
- **Health Checks**: Every 10s with 3 unhealthy threshold
- **Circuit Breakers**: 1000 max connections, 3 max retries
- **Outlier Detection**: Ejects hosts after 5 consecutive 5xx errors

#### Retry and Timeout Configuration

```mermaid
sequenceDiagram
    participant C as Client
    participant E as Envoy
    participant W1 as Web Service 1
    participant W2 as Web Service 2

    Note over E: Overall Timeout: 30s<br/>Per-Try Timeout: 10s<br/>Max Retries: 3

    C->>E: Request

    rect rgb(240, 240, 255)
        Note over E,W1: Attempt 1
        E->>W1: Forward (10s timeout)
        W1--xE: 500 Internal Server Error
        Note over E: Retry conditions met:<br/>5xx error
    end

    rect rgb(255, 240, 240)
        Note over E,W2: Attempt 2 (Different Host)
        E->>W2: Forward (10s timeout)
        W2--xE: Connection Timeout
        Note over E: Retry conditions met:<br/>Connection failure
    end

    rect rgb(255, 250, 240)
        Note over E,W1: Attempt 3 (Back to W1)
        E->>W1: Forward (10s timeout)
        W1--xE: Refused Stream
        Note over E: Retry conditions met:<br/>Refused stream
    end

    rect rgb(240, 255, 240)
        Note over E,W2: Attempt 4 (Final)
        E->>W2: Forward (10s timeout)
        W2-->>E: 200 OK + Response
        Note over E: Success!
    end

    E-->>C: 200 OK + Response

    Note over C,W2: Total elapsed: ~32s<br/>(4 attempts × 10s each - some faster)
```

**Configuration Details:**
- **Total Request Timeout**: 30 seconds maximum
- **Per-Try Timeout**: 10 seconds per attempt
- **Retry On**: 5xx errors, connection failures, refused streams
- **Max Retries**: 3 (4 total attempts)
- **Host Selection**: Avoids previously failed hosts when possible

## Advanced Usage

### View Service Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f web-service-1
docker-compose logs -f envoy-proxy
```

### Scale Services

```bash
# Add more instances
docker-compose up --scale web-service-1=2 -d
```

### Test Failure Scenarios

```bash
# Stop one service to test health checks
docker-compose stop web-service-1

# Requests will be routed to healthy instances
curl http://localhost:10000
```

### Monitor with Envoy Admin

- **Stats**: http://localhost:9901/stats
- **Clusters**: http://localhost:9901/clusters
- **Server Info**: http://localhost:9901/server_info
- **Config Dump**: http://localhost:9901/config_dump

## Development

### Multi-Stage Build Process

```mermaid
graph LR
    subgraph "Build Stage (golang:1.23-alpine)"
        A[Copy go.mod/go.sum] --> B[Download Dependencies]
        B --> C[Copy Source Code]
        C --> D[Build Static Binary]
        D --> E[web-service binary<br/>~5-8MB]
    end

    subgraph "Runtime Stage (scratch)"
        F[Empty Base Image] --> G[Copy CA Certificates]
        G --> H[Copy Timezone Data]
        H --> I[Copy Binary]
        I --> J[Final Image<br/>~15MB]
    end

    E -.->|copy| I

    style E fill:#bfb,stroke:#333,stroke-width:2px
    style J fill:#bbf,stroke:#333,stroke-width:3px
    style F fill:#fbb,stroke:#333,stroke-width:2px
```

**Benefits:**
- **95% size reduction**: 300MB → 15MB
- **Layer caching**: Dependencies cached separately
- **Security**: Minimal attack surface with scratch base
- **Performance**: Faster deployments and startup

### Graceful Shutdown Flow

```mermaid
sequenceDiagram
    participant OS as Operating System
    participant Main as Main Goroutine
    participant Server as HTTP Server
    participant Req1 as Active Request 1
    participant Req2 as Active Request 2

    Note over Server: Server Running

    Req1->>Server: GET / (processing)
    Req2->>Server: GET /health (processing)

    OS->>Main: SIGTERM/SIGINT
    Note over Main: Signal Received

    Main->>Server: Shutdown(30s timeout)
    Note over Server: Stop Accepting<br/>New Connections

    rect rgb(255, 200, 200)
        Note over Server: New Requests Rejected
    end

    Server->>Req1: Continue Processing
    Server->>Req2: Continue Processing

    Req1-->>Server: Response Completed
    Req2-->>Server: Response Completed

    Server-->>Main: Shutdown Complete
    Note over Main: Exit(0)

    alt Timeout Exceeded
        Note over Server: Force Close After 30s
        Server-xReq1: Connection Closed
        Server-xReq2: Connection Closed
        Server-->>Main: Shutdown Error
        Note over Main: Exit(1)
    end
```

### Build Individual Services

```bash
# Go service
cd go-web-service
docker build -t web-service .

# Envoy proxy
cd envoy-proxy
docker build -t envoy-proxy .
```

### Run Tests

```bash
# Test Go service directly
cd go-web-service
go test ./...
```

## Production Considerations

This demo includes production-ready patterns:
- ✅ Graceful shutdown handlers
- ✅ Health check endpoints
- ✅ Structured logging
- ✅ Resource limits
- ✅ Non-root containers
- ✅ Multi-stage builds
- ✅ Retry and circuit breaking policies

For production deployment, consider adding:
- TLS/mTLS for service-to-service communication
- Distributed tracing (OpenTelemetry)
- Metrics collection (Prometheus)
- Service mesh integration (Istio/Linkerd)

## Troubleshooting

**Services won't start:**
- Check Docker daemon is running
- Ensure ports 10000 and 9901 are available

**Health checks failing:**
- Check service logs: `docker-compose logs web-service-1`
- Verify `/health` endpoint: `docker exec <container> wget -O- localhost:1337/health`

**No load balancing:**
- Verify all services are healthy: `curl localhost:9901/clusters`
- Check Envoy logs: `docker-compose logs envoy-proxy`

## References

- [Envoy Documentation](https://www.envoyproxy.io/docs)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Go HTTP Server Best Practices](https://golang.org/doc/articles/wiki/)
