package blog

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	log.Printf("Началао gRPC запроса: %s", info.FullMethod)
	response, err := handler(ctx, req)

	log.Printf(
		"Конец gRPC запроса %s | время: %s | код: %s",
		info.FullMethod,
		time.Since(start),
		status.Code(err),
	)

	return response, err
}
