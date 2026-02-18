# gRPC Examples in Go

A collection of gRPC examples in Go, demonstrating different RPC patterns with observability (logging + Prometheus metrics).

## Examples

| Example | Description |
|---|---|
| [Unary RPC](./unary/README.md) | Single request / single response |

## Prerequisites

- Go 1.23+
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` and `protoc-gen-go-grpc` plugins

## Project Structure

```
grpc/
├── go.mod / go.sum
└── unary/               # Unary RPC example
    ├── Makefile         # Generates proto stubs
    ├── pb/
    │   ├── device.proto
    │   ├── device.pb.go
    │   └── device_grpc.pb.go
    ├── server/server.go
    └── client/client.go
```

## Generating Proto Stubs

```bash
cd unary
make
```

This runs `protoc` to regenerate the Go stubs from `device.proto`.

## Running the Unary Example

Start the server:
```bash
go run unary/server/server.go
```

In a separate terminal, start the client:
```bash
go run unary/client/client.go
```

The client sends 10 requests to the server (one every 10 seconds).

## Observability

| Component | Endpoint | Description |
|---|---|---|
| Server | `:9092/metrics` | Server-side gRPC Prometheus metrics |
| Client | `:9094/metrics` | Client-side gRPC Prometheus metrics |

## Key Concepts

- **Unary RPC** — simplest RPC pattern; one request, one response
- **Interceptors** — middleware chain for logging (logrus) and metrics (Prometheus) on both client and server
- **Keepalive** — persistent HTTP/2 connections to avoid TCP setup overhead per call
- **Protocol Buffers** — schema-first service definition and binary serialization
