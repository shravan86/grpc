package main

import (
    "context"
    "net"
    "net/http"
    "shravan/grpc/unary/pb"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/sirupsen/logrus"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/keepalive"
    "google.golang.org/grpc/status"

    "github.com/grpc-ecosystem/go-grpc-prometheus"
)

var (
    log = logrus.NewEntry(logrus.New())

    // grpcMetrics collects all the default server gRPC metrics.
    grpcMetrics = grpc_prometheus.NewServerMetrics()
)

// server is used to implement pb.DeviceInfoServer.
type server struct {
}

// GetDevice implements pb.DeviceInfoServer
// Returns device information for a given device id (serial or mac address).
func (*server) GetDevice(ctx context.Context, req *pb.DeviceRequest) (*pb.Device, error) {
    if req.GetSerial() == "" && req.GetMacAddr() == "" {
        return nil, status.Errorf(codes.InvalidArgument, "device_id is required")
    }

    return &pb.Device{
        Serial:    "A",
        MacAddr:   "00:00:00:00:00:00",
        Type:      pb.DeviceType_AP,
        Connected: true,
    }, nil
}

func main() {
    var (
        err error
        lis net.Listener
    )

    s := initGrpcServer()

    // Registers and Initializes all the default gRPC server metrics and rpcCounter.
    go startPromServer(s)

    pb.RegisterDeviceInfoServer(s, &server{})

    if lis, err = net.Listen("tcp", ":8080"); err != nil {
        panic(err)
    }

    if err = s.Serve(lis); err != nil {
        panic(err)
    }
}

func initGrpcServer() *grpc.Server {
    return grpc.NewServer(
        // Adds logrus interceptor to log the requests and responses.
        // Adds prometheus interceptor to collect metrics.
        grpc.ChainUnaryInterceptor(loggingInterceptor, grpcMetrics.UnaryServerInterceptor()),

        // Enables keepalive pings to be sent to the client.
        // This is required to make sure that the connection is not teared down.
        // Default timeout is 30 mins. Setting up a new TCP connection for each
        // rpc call is expensive and defeats the purpose of HTTP2.
        grpc.KeepaliveParams(keepalive.ServerParameters{
            // Closes connection after 15 minutes of inactivity.
            MaxConnectionIdle: 15 * time.Minute,

            // Pings the client every 10 seconds if it is idle.
            Time: 10 * time.Second,

            // Waits 5 secs for ping response.
            Timeout: 5 * time.Second,
        }),

        // Allows pings from client even when there are no active streams.
        grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
            // Minimum time between client pings.
            MinTime: 20 * time.Second,

            // Allow pings without active streams.
            PermitWithoutStream: true,
        }),
    )
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
    var (
        resp interface{}
        err  error
    )

    log.Infof("gRPC call: %v req: %v", info.FullMethod, req)

    start := time.Now()
    if resp, err = handler(ctx, req); err != nil {
        log.Errorf("gRPC call: %v error: %v", info.FullMethod, err)

        return nil, err
    }

    log.Infof("gRPC call: %v duration: %v reply: %v", info.FullMethod, time.Since(start), resp)

    return resp, nil
}

func startPromServer(s *grpc.Server) {
    // Registers and Initializes all the default gRPC server metrics and rpcCounter.
    reg := prometheus.NewRegistry()
    reg.MustRegister(grpcMetrics)
    grpcMetrics.InitializeMetrics(s)

    httpServer := &http.Server{
        Addr:    ":9092",
        Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
    }

    if err := httpServer.ListenAndServe(); err != nil {
        panic(err)
    }
}
