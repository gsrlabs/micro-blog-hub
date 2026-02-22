package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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

type PostLike struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	PostID    primitive.ObjectID `bson:"post_id"`
	UserID    string             `bson:"user_id"`
	CreatedAt time.Time          `bson:"created_at"`
}

type Comment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	PostID    primitive.ObjectID `bson:"post_id"`
	AuthorID  string             `bson:"author_id"`
	Content   string             `bson:"content"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusHidden    PostStatus = "hidden"
	PostStatusDeleted   PostStatus = "deleted"
)
