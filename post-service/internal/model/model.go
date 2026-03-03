package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusHidden    PostStatus = "hidden"
	PostStatusDeleted   PostStatus = "deleted"
)

type Post struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID      string             `bson:"author_id"`
	Title         string             `bson:"title"`
	Content       string             `bson:"content"`
	Topic         string             `bson:"topic,omitempty"`
	Tags          []string           `bson:"tags,omitempty"`
	LikesCount    int64              `bson:"likes_count"`
	Views         int64              `bson:"views"`
	CommentsCount int64              `bson:"comments_count"`
	Slug          string             `bson:"slug"`
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
	DeletedAt     *time.Time         `bson:"deleted_at,omitempty"`
	Status        PostStatus         `bson:"status"`
}

type PaginatedPosts struct {
	Items []*Post
	Total int64
	Page  int64
	Limit int64
}

type PaginatedPostsWithLikeState struct {
	Items []*PostWithLikeState `json:"items"`
	Total int64                `json:"total"`
	Page  int64                `json:"page"`
	Limit int64                `json:"limit"`
}

type PostWithLikeState struct {
	Post    *Post
	IsLiked bool
}

type PostLike struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	PostID    primitive.ObjectID `bson:"post_id"`
	UserID    string             `bson:"user_id"`
	CreatedAt time.Time          `bson:"created_at"`
}

type Comment struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	PostID     primitive.ObjectID `bson:"post_id"`
	AuthorID   string             `bson:"author_id"`
	Content    string             `bson:"content"`
	LikesCount int64              `bson:"likes_count"`
	CreatedAt  time.Time          `bson:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at"`
}

type ListCommentsResult struct {
	Comments   []*Comment
	TotalCount int64
}

type CommentLike struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CommentID primitive.ObjectID `bson:"comment_id"`
	UserID    string             `bson:"user_id"`
	CreatedAt time.Time          `bson:"created_at"`
}

type CommentWithLikeState struct {
	Comment *Comment `bson:"comment"`
	IsLiked bool     `bson:"is_liked"`
}
