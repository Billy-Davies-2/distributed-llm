package network

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// MetricsInterceptor provides gRPC interceptors for metrics collection
type MetricsInterceptor struct {
	metricsCollector MetricsCollector
}

// NewMetricsInterceptor creates a new metrics interceptor
func NewMetricsInterceptor(collector MetricsCollector) *MetricsInterceptor {
	return &MetricsInterceptor{
		metricsCollector: collector,
	}
}

// UnaryServerInterceptor returns a new unary server interceptor for metrics
func (mi *MetricsInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req) // Record metrics
		duration := time.Since(start)

		// Determine status string for metrics
		statusStr := "success"
		if err != nil {
			statusStr = "error"
		}

		// Extract method name from full method
		method := info.FullMethod

		// Record inference request if it's an inference method
		if method == "/node.NodeService/ProcessInference" {
			tokensGenerated := 10 // Default mock value - in real implementation extract from response
			if mi.metricsCollector != nil {
				mi.metricsCollector.RecordInferenceRequest("default_model", statusStr, duration, tokensGenerated)
			}
		}

		// Record general network latency
		if mi.metricsCollector != nil {
			mi.metricsCollector.RecordNetworkLatency("grpc_client", method, duration)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a new stream server interceptor for metrics
func (mi *MetricsInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, ss)

		// Record metrics
		duration := time.Since(start)

		// Record network latency for streaming operations
		if mi.metricsCollector != nil {
			mi.metricsCollector.RecordNetworkLatency("grpc_client", info.FullMethod, duration)
		}

		return err
	}
}
