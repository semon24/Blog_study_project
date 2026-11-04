package blog

import (
	"context"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func userIDFromContext(ctx context.Context) (int64, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("x-user-id")
	if len(values) == 0 {
		return 0, status.Error(codes.Unauthenticated, "x-user-id is required")
	}
	if len(values) != 1 {
		return 0, status.Error(codes.InvalidArgument, "provide exactly one x-user-id")
	}

	userID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil || userID <= 0 {
		return 0, status.Error(codes.InvalidArgument, "x-user-id must be a positive integer")
	}

	return userID, nil
}
