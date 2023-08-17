# Edge Proxy Demo with Docker

In this demo, we will set up Envoy as an edge proxy using Docker. Envoy will handle incoming traffic for a local Go web service also running in a Docker container.

## Prerequisites

- Docker installed on your local machine.
- Go installed on your local machine.

## Steps

1. **Create a simple Go web service.**
   Create a file named `main.go` with the following content:
   ```go
   package main

   import (
       "fmt"
       "net/http"
   )

   func main() {
       http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
           fmt.Fprintf(w, "Hello, World!")
       })

       http.ListenAndServe(":8080", nil)
   }
   ```

2. **Containerize the Go web service using Docker.**

   Create a file named `Dockerfile` with the following content:
   ```dockerfile
   FROM golang:1.21.0-alpine3.18

   WORKDIR /app

   COPY main.go .

   RUN go env -w GO111MODULE=auto && go build -o main .

   EXPOSE 8080

   CMD ["/app/main"]
   ```
   Build the Docker image:
   ```bash
   docker build -t go-web-service .
   ```
   Run the Docker container:
   ```bash
   docker run -d -p 8080:8080 go-web-service
   ```
   Test the web service by visiting http://localhost:8080 in your browser.

3. **Pull the official Envoy Docker image.**

   ```bash
   docker pull envoyproxy/envoy:v1.27-latest
   ```

4. **Create a Dockerfile with a custom envoy.yaml configuration file.**
The `envoy.yaml` file should be configured to route incoming traffic to the Go web service. Here's a basic example:

```yaml
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address: { address: 0.0.0.0, port_value: 10000 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: { host_rewrite_literal: "localhost", cluster: service_go }
          http_filters:
          - name: envoy.filters.http.router
  clusters:
  - name: service_go
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: service_go
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: host.docker.internal
                port_value: 8080
```

Create a Dockerfile for Envoy:

```dockerfile
FROM envoyproxy/envoy:v1.27-latest
COPY envoy.yaml /etc/envoy/envoy.yaml
RUN chmod go+r /etc/envoy/envoy.yaml
```

5. **Build the Envoy Docker image.**

```bash
docker build -t envoy:demo .
```

6. **Run the Envoy Docker container.**

```bash
docker run -d --name envoy --rm -p 9901:9901 -p 10000:10000 envoy:demo
```

7. **Test the setup by sending requests to the Go web service via the Envoy proxy.**

```bash
curl -v http://localhost:10000
```
 You should something similar to the following output:
 ```
*   Trying 127.0.0.1:10000...
* Connected to localhost (127.0.0.1) port 10000 (#0)
> GET / HTTP/1.1
> Host: localhost:10000
> User-Agent: curl/8.1.2
> Accept: */*
> 
< HTTP/1.1 200 OK
< date: Thu, 17 Aug 2023 23:07:24 GMT
< content-length: 31
< content-type: text/plain; charset=utf-8
< x-envoy-upstream-service-time: 16
< server: envoy
< 
* Connection #0 to host localhost left intact
Hello, World! From 5f1961e57b27
```

### Notes: 
The `envoy.yaml` file is configured to route incoming traffic to the Go web service running on port `8080`. The `host.docker.internal` hostname is used to refer to the host machine from within the Docker container.
The `curl` command is used to send a request to the Go web service via the Envoy proxy. You should see the response from the Go web service.
