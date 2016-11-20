// Package grpc provides a gRPC client for the add service.
package grpc

import (
	//"fmt"
	"time"

	jujuratelimit "github.com/juju/ratelimit"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/tracing/opentracing"		
	grpctransport "github.com/go-kit/kit/transport/grpc"
	
	"fault_injection/fi"
	"fault_injection/pb"
)

// New returns an AddService backed by a gRPC client connection. It is the
// responsibility of the caller to dial, and later close, the connection.
func New(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) fi.Service {
	// We construct a single ratelimiter middleware, to limit the total outgoing
	// QPS from this client to all methods on the remote instance. We also
	// construct per-endpoint circuitbreaker middlewares to demonstrate how
	// that's done, although they could easily be combined into a single breaker
	// for the entire remote instance, too.

	limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))

	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = grpctransport.NewClient(
			conn,
			"Add",
			"Sum",
			fi.EncodeGRPCSumRequest,
			fi.DecodeGRPCSumResponse,
			pb.SumReply{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		sumEndpoint = opentracing.TraceClient(tracer, "Sum")(sumEndpoint)
		sumEndpoint = limiter(sumEndpoint)
		sumEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Sum",
			Timeout: 30 * time.Second,
		}))(sumEndpoint)
	}

	var concatEndpoint endpoint.Endpoint
	{
		concatEndpoint = grpctransport.NewClient(
			conn,
			"Add",
			"Concat",
			fi.EncodeGRPCConcatRequest,
			fi.DecodeGRPCConcatResponse,
			pb.ConcatReply{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		concatEndpoint = opentracing.TraceClient(tracer, "Concat")(concatEndpoint)
		concatEndpoint = limiter(concatEndpoint)
		concatEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Concat",
			Timeout: 30 * time.Second,
		}))(concatEndpoint)
	}

	return fi.Endpoints{
		SumEndpoint:    sumEndpoint,
		ConcatEndpoint: concatEndpoint,
	}
}