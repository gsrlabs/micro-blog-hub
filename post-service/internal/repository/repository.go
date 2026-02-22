package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gsrlabs/micro-blog-hub/post-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	GetByID(ctx context.Context, id string) (*model.Post, error)
	Update(ctx context.Context, post *model.Post) error
	Delete(ctx context.Context, id string) error
	FindByTag(ctx context.Context, tag string) ([]*model.Post, error)
	FindByTopic(ctx context.Context, topic string) ([]*model.Post, error)
}

type postRepo struct {
	client *mongo.Client
	dbname string
	logger *zap.Logger
}

var (
	ErrNotFound   = errors.New("post not found")
	ErrSlugExists = errors.New("slug already exists")
)

func NewPostRepository(client *mongo.Client, dbName string, logger *zap.Logger) PostRepository {
	repo := &postRepo{
		client: client,
		dbname: dbName,
		logger: logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.ensureIndexes(ctx); err != nil {
		logger.Fatal("failed to create indexes", zap.Error(err))
	}

	return repo
}

func (r *postRepo) collection() *mongo.Collection {
	return r.client.Database(r.dbname).Collection("posts")
}

func (r *postRepo) Create(ctx context.Context, post *model.Post) error {
	post.ID = primitive.NewObjectID()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	_, err := r.collection().InsertOne(ctx, post)
	if err != nil {

		if mongo.IsDuplicateKeyError(err) {
			r.logger.Warn("slug already exists",
				zap.String("slug", post.Slug),
			)
			return ErrSlugExists
		}

		r.logger.Error("failed to insert post",
			zap.Error(err),
		)
		return err
	}

	r.logger.Info("post created",
		zap.String("post_id", post.ID.Hex()),
		zap.String("slug", post.Slug),
	)

	return nil
}

func (r *postRepo) ensureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"slug": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.collection().Indexes().CreateOne(ctx, indexModel)
	return err
}

func (r *postRepo) GetByID(ctx context.Context, id string) (*model.Post, error) {

	// 1️⃣ Конвертируем string → ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", id),
		)
		return nil, ErrNotFound
	}

	// 2️⃣ Фильтр
	filter := bson.M{"_id": objectID}

	var post model.Post

	// 3️⃣ Поиск
	err = r.collection().FindOne(ctx, filter).Decode(&post)
	if err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.Warn("post not found",
				zap.String("post_id", id),
			)
			return nil, ErrNotFound
		}

		r.logger.Error("failed to get post",
			zap.Error(err),
			zap.String("post_id", id),
		)
		return nil, err
	}

	return &post, nil
}

func (r *postRepo) Update(ctx context.Context, post *model.Post) error
func (r *postRepo) Delete(ctx context.Context, id string) error
func (r *postRepo) FindByTag(ctx context.Context, tag string) ([]*model.Post, error)
func (r *postRepo) FindByTopic(ctx context.Context, topic string) ([]*model.Post, error)
