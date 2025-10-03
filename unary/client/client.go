package main

import (
    "context"
    "net/http"
    "shravan/grpc/unary/pb"
    "time"

    grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/sirupsen/logrus"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"
)

var (
    // grpcMetrics collects all the default client gRPC metrics.
    grpcMetrics = grpcProm.NewClientMetrics()
)

func main() {
    var (
        err  error
        conn *grpc.ClientConn
    )

    // Sets logging format.
    logger := logrus.New()
    logger.SetFormatter(&logrus.TextFormatter{
        TimestampFormat: "2006-01-02 15:04:05.000", // Includes milliseconds
        FullTimestamp:   true,
    })

    opts := initGrpcConfig(logger)

    if conn, err = grpc.NewClient(":8080", opts...); err != nil {
        panic(err)
    }

    cl := pb.NewDeviceInfoClient(conn)

    go startPromServer()

    // Sends 10 requests with an interval of 10 seconds.
    for i := 0; i < 10; i++ {
        sendRequest(cl)
        time.Sleep(10 * time.Second)
    }
}

func initGrpcConfig(logger *logrus.Logger) []grpc.DialOption {
    return []grpc.DialOption{
        // Insecure connection for the sake of simplicity.
        // We will not use TLS handshake for this example.
        grpc.WithTransportCredentials(insecure.NewCredentials()),

        // Adds logrus interceptor to log the requests and responses.
        // Adds prometheus interceptor to collect gRPC client metrics.
        grpc.WithChainUnaryInterceptor(
            loggingInterceptor(logger),
            grpcMetrics.UnaryClientInterceptor(),
        ),

        // Enables keepalive pings to be sent to the server.
        // It is required to make sure that the connection is not tear down.
        // Setting up a new TCP connection for each rpc call is expensive and defeats the purpose of HTTP2.
        // Default timeout is 30 mins.
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            // Sends keepalive pings every 10 seconds.
            Time: 10 * time.Second,

            // Waits 5 secs for ping ack before considering the connection dead.
            Timeout: 5 * time.Second,

            // Sends pings even when no active RPCs.
            PermitWithoutStream: true,
        }),
    }
}

// loggingInterceptor returns a new unary client interceptor that logs the RPC call.
func loggingInterceptor(logger *logrus.Logger) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        logger.Infof("gRPC call: %s req: %v", method, req)

        start := time.Now()
        err := invoker(ctx, method, req, reply, cc, opts...)
        duration := time.Since(start)

        if err != nil {
            logger.Errorf("gRPC call: %v error: %v duration: %v", method, err, duration)
        } else {
            logger.Infof("gRPC call: %v duration: %v reply: %v", method, duration, reply)
        }

        return err
    }
}

func startPromServer() {
    // Registers and Initializes all the default gRPC server metrics and rpcCounter.
    reg := prometheus.NewRegistry()
    reg.MustRegister(grpcMetrics)

    httpServer := &http.Server{
        Addr:    ":9094",
        Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
    }

    if err := httpServer.ListenAndServe(); err != nil {
        panic(err)
    }
}

func sendRequest(cl pb.DeviceInfoClient) {
    // Builds request.
    req := &pb.DeviceRequest{
        DeviceId: &pb.DeviceRequest_Serial{
            Serial: "A",
        },
    }

    if _, err := cl.GetDevice(context.Background(), req); err != nil {
        panic(err)
    }
}
