package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/metadata"
	"io"
	"math"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Logger interface {
	Info(args ...interface{})
}

type RPCLog struct {
	ID           string `json:"id"`
	StartTime    string `json:"startTime"`
	ResponseTime int64  `json:"responseTime"`
	Method       string `json:"method"`
	StatusCode   int32  `json:"statusCode"`
}

func (l RPCLog) PrettyPrint(writer io.Writer) {
	// checking the length of status code to match the spacing that is being done in HTTP logs after status codes
	statusCodeLen := 9 - int(math.Log10(float64(l.StatusCode))) + 1

	fmt.Fprintf(writer, "\u001B[38;5;8m%s \u001B[38;5;%dm%-6d"+
		"\u001B[0m %*d\u001B[38;5;8mµs\u001B[0m %s \n",
		l.ID, colorForGRPCCode(l.StatusCode),
		l.StatusCode, statusCodeLen, l.ResponseTime, l.Method)
}

func colorForGRPCCode(s int32) int {
	const (
		blue = 34
		red  = 202
	)

	if s == 0 {
		return blue
	}

	return red
}

func (l RPCLog) String() string {
	line, _ := json.Marshal(l)
	return string(line)
}

func LoggingInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Retrieve metadata from the incoming context
		md, ok := metadata.FromIncomingContext(ctx)
		var parentSpanContext trace.SpanContext
		var span trace.Span

		if ok {
			// Extract the serialized span context (if present) from metadata
			spanContextStr := md.Get("parent-span-context")
			if len(spanContextStr) > 0 {
				parentSpanContext = spanContextFromString(spanContextStr[0])
			}
		}

		// Check if a valid parent span context exists
		if parentSpanContext.IsValid() {
			// Create a new child span from the parent span context
			tracectx := trace.ContextWithSpanContext(ctx, parentSpanContext)
			ctx, span = otel.GetTracerProvider().
				Tracer("gofr", trace.WithInstrumentationVersion("v0.1")).
				Start(tracectx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		} else {
			// Create a new root span
			ctx, span = otel.GetTracerProvider().
				Tracer("gofr", trace.WithInstrumentationVersion("v0.1")).
				Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))

			// Add the serialized span context to metadata for propagation
			serializedSpanContext := spanContextToString(span.SpanContext())
			md = metadata.Pairs("parent-span-context", serializedSpanContext)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		start := time.Now()

		// Call the handler with the updated context
		resp, err := handler(ctx, req)

		// Defer the logging and span end
		defer func() {
			l := RPCLog{
				ID:           span.SpanContext().TraceID().String(),
				StartTime:    start.Format("2006-01-02T15:04:05.999999999-07:00"),
				ResponseTime: time.Since(start).Microseconds(),
				Method:       info.FullMethod,
			}

			// Log the status code based on the error
			if err != nil {
				if statusErr, ok := status.FromError(err); ok {
					l.StatusCode = int32(statusErr.Code())
				}
			} else {
				l.StatusCode = int32(codes.OK)
			}

			if logger != nil {
				logger.Info(l)
			}

			// End the span
			span.End()
		}()

		return resp, err
	}
}

// Helper function to serialize a SpanContext to a string
func spanContextToString(sc trace.SpanContext) string {
	return fmt.Sprintf("%s|%s", sc.TraceID().String(), sc.SpanID().String())
}

// Helper function to deserialize a SpanContext from a string
func spanContextFromString(str string) trace.SpanContext {
	parts := strings.Split(str, "|")
	if len(parts) != 2 {
		return trace.SpanContext{}
	}

	traceID, err := trace.TraceIDFromHex(parts[0])
	if err != nil {
		return trace.SpanContext{}
	}

	spanID, err := trace.SpanIDFromHex(parts[1])
	if err != nil {
		return trace.SpanContext{}
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	})
}
