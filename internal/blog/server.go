package blog

import (
	blogpb "blog-study-project/gen/blog/v1"
	"blog-study-project/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const firstPageCacheKey = "posts:first-page"

type Server struct {
	blogpb.UnimplementedBlogServiceServer
	DB      *gorm.DB
	Redis   *redis.Client
	postsMu sync.RWMutex
}

func (s *Server) loadPosts(ctx context.Context, req *blogpb.ListPostsRequest) ([]storage.Post, error) {
	useCache := req.Offset == 0 && req.Limit == 10
	if useCache {
		s.postsMu.RLock()
		defer s.postsMu.RUnlock()
	}

	var posts []storage.Post
	if useCache {
		data, err := s.Redis.Get(ctx, firstPageCacheKey).Bytes()
		if err == nil {
			if err := json.Unmarshal(data, &posts); err == nil {
				return posts, nil
			}
			posts = nil
		} else if !errors.Is(err, redis.Nil) {
			log.Printf("Read posts cache: %v", err)
		}
	}

	query := s.DB.WithContext(ctx).
		Preload("Author").
		Order("created_at DESC, id DESC").
		Offset(int(req.Offset))
	if req.Limit > 0 {
		query = query.Limit(int(req.Limit))
	}
	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}
	if useCache {
		data, err := json.Marshal(posts)
		if err != nil {
			log.Printf("Encode posts cache: %v", err)
		} else if err := s.Redis.Set(ctx, firstPageCacheKey, data, time.Minute).Err(); err != nil {
			log.Printf("Write posts cache: %v", err)
		}
	}
	return posts, nil
}

func (s *Server) invalidatePostsCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Redis.Del(ctx, firstPageCacheKey).Err(); err != nil {
		log.Printf("Invalidate posts cache: %v", err)
	}
}

func (s *Server) ListPosts(
	ctx context.Context,
	req *blogpb.ListPostsRequest,
) (*blogpb.ListPostsResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Limit < 0 || req.Offset < 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"limit and offset must be non-negative",
		)
	}

	posts, err := s.loadPosts(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list posts")
	}

	likesCounts := make([]*redis.IntCmd, len(posts))
	likedByMe := make([]*redis.BoolCmd, len(posts))
	if len(posts) > 0 {
		_, err := s.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, post := range posts {
				key := "post:" + strconv.FormatInt(post.ID, 10) + ":likes"
				likesCounts[i] = pipe.SCard(ctx, key)
				likedByMe[i] = pipe.SIsMember(ctx, key, userID)
			}
			return nil
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to load post likes")
		}
	}

	result := make([]*blogpb.Post, 0, len(posts))

	for i, post := range posts {
		result = append(result, &blogpb.Post{
			Id: post.ID,
			Author: &blogpb.Author{
				Id:       post.Author.ID,
				Nickname: post.Author.Nickname,
				PhotoUrl: post.Author.PhotoURL,
			},
			Body:       post.Body,
			CreatedAt:  timestamppb.New(post.CreatedAt),
			LikesCount: likesCounts[i].Val(),
			LikedByMe:  likedByMe[i].Val(),
		})
	}

	return &blogpb.ListPostsResponse{
		Posts: result,
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

	author := storage.Author{
		ID:       userID,
		Nickname: "jack",
		PhotoURL: "https://example.com/avatar.jpg",
	}

	post := storage.Post{
		AuthorID: userID,
		Body:     req.Body,
	}

	s.postsMu.Lock()
	defer s.postsMu.Unlock()

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&author).Error; err != nil {
			return err
		}

		if err := tx.First(&author, userID).Error; err != nil {
			return err
		}

		return tx.Create(&post).Error
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create post")
	}
	s.invalidatePostsCache()

	return &blogpb.CreatePostResponse{
		Post: &blogpb.Post{
			Id: post.ID,
			Author: &blogpb.Author{
				Id:       author.ID,
				Nickname: author.Nickname,
				PhotoUrl: author.PhotoURL,
			},
			Body:      post.Body,
			CreatedAt: timestamppb.New(post.CreatedAt),
		},
	}, nil
}

func (s *Server) UpdatePost(
	ctx context.Context,
	req *blogpb.UpdatePostRequest,
) (*blogpb.UpdatePostResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.PostId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "post_id must be positive")
	}

	var post storage.Post
	s.postsMu.Lock()
	defer s.postsMu.Unlock()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&post).
			Clauses(clause.Returning{}).
			Where("id = ? AND author_id = ?", req.PostId, userID).
			Update("body", req.Body)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.First(&post.Author, post.AuthorID).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "post not found or belongs to another user")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update post")
	}
	s.invalidatePostsCache()

	return &blogpb.UpdatePostResponse{
		Post: &blogpb.Post{
			Id: post.ID,
			Author: &blogpb.Author{
				Id:       post.Author.ID,
				Nickname: post.Author.Nickname,
				PhotoUrl: post.Author.PhotoURL,
			},
			Body:      post.Body,
			CreatedAt: timestamppb.New(post.CreatedAt),
		},
	}, nil
}

func (s *Server) DeletePost(
	ctx context.Context,
	req *blogpb.DeletePostRequest,
) (*blogpb.DeletePostResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.PostId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "post_id must be positive")
	}

	s.postsMu.Lock()
	defer s.postsMu.Unlock()
	result := s.DB.WithContext(ctx).
		Where("id = ? AND author_id = ?", req.PostId, userID).
		Delete(&storage.Post{})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, "failed to delete post")
	}

	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "post not found or belongs to another user")
	}
	s.invalidatePostsCache()

	return &blogpb.DeletePostResponse{}, nil
}

func (s *Server) SetPostLike(
	ctx context.Context,
	req *blogpb.SetPostLikeRequest,
) (*blogpb.SetPostLikeResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.PostId <= 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"post_id must be positive",
		)
	}

	var post storage.Post
	err = s.DB.WithContext(ctx).
		Select("id").
		First(&post, req.PostId).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "post not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to find post")
	}

	key := "post:" + strconv.FormatInt(req.PostId, 10) + ":likes"

	if req.Liked {
		err = s.Redis.SAdd(ctx, key, userID).Err()
	} else {
		err = s.Redis.SRem(ctx, key, userID).Err()
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to set post like")
	}

	return &blogpb.SetPostLikeResponse{}, nil
}
