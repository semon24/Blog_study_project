package blog

import (
	blogpb "blog-study-project/gen/blog/v1"
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	blogpb.UnimplementedBlogServiceServer
}

func (s *Server) ListPosts(
	ctx context.Context,
	req *blogpb.ListPostsRequest,
) (*blogpb.ListPostsResponse, error) {
	_, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Limit < 0 || req.Offset < 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"limit and offset must be positive integers")
	}

	post := &blogpb.Post{
		Id: 1,
		Author: &blogpb.Author{
			Id:       42,
			Nickname: "jack",
			PhotoUrl: "https://example.com/avatar.jpg",
		},
		Body: "Мой первый пост",
		CreatedAt: timestamppb.New(
			time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC),
		),
		LikesCount: 5,
		LikedByMe:  false,
	}

	posts := []*blogpb.Post{post}
	if req.Offset >= int64(len(posts)) {
		return &blogpb.ListPostsResponse{}, nil
	}
	posts = posts[req.Offset:]

	if req.Limit > int64(len(posts)) || req.Limit == 0 {
		posts = posts[:int64(len(posts))]
	} else {
		posts = posts[:req.Limit]
	}

	return &blogpb.ListPostsResponse{
		Posts: posts,
	}, nil
}

func (s *Server) CreatePost(
	ctx context.Context,
	req *blogpb.CreatePostRequest,
) (*blogpb.CreatePostResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	post := &blogpb.Post{
		Id: 2,
		Author: &blogpb.Author{
			Id:       userID,
			Nickname: "jack",
			PhotoUrl: "https://example.com/avatar.jpg",
		},
		Body:      req.Body,
		CreatedAt: timestamppb.Now(),
	}

	return &blogpb.CreatePostResponse{
		Post: post,
	}, nil
}

func (s *Server) UpdatePost(
	ctx context.Context,
	req *blogpb.UpdatePostRequest,
) (*blogpb.UpdatePostResponse, error) {
	_, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	post := &blogpb.Post{
		Id: req.PostId,
		Author: &blogpb.Author{
			Id:       42,
			Nickname: "jack",
			PhotoUrl: "https://example.com/avatar.jpg",
		},
		Body: req.Body,
		CreatedAt: timestamppb.New(
			time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC),
		),
		LikesCount: 5,
		LikedByMe:  false,
	}

	return &blogpb.UpdatePostResponse{
		Post: post,
	}, nil
}

func (s *Server) DeletePost(
	ctx context.Context,
	req *blogpb.DeletePostRequest,
) (*blogpb.DeletePostResponse, error) {
	_, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &blogpb.DeletePostResponse{}, nil
}

func (s *Server) SetPostLike(
	ctx context.Context,
	req *blogpb.SetPostLikeRequest,
) (*blogpb.SetPostLikeResponse, error) {
	_, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &blogpb.SetPostLikeResponse{}, nil
}
