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

var (
	ErrNotFound   = errors.New("post not found")
	ErrSlugExists = errors.New("slug already exists")
)

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	GetByID(ctx context.Context, id string) (*model.Post, error)
	GetBySlug(ctx context.Context, slug string) (*model.Post, error)
	Update(ctx context.Context, post *model.Post) error
	MarkAsDeleted(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	ListPostsAdvanced(
		ctx context.Context,
		userID string,
		topic string,
		tag string,
		sortBy string,
		sortOrder int,
		page, limit int64,
	) (*model.PaginatedPostsWithLikeState, error)
	IncrementViews(ctx context.Context, id string) error
	AddLike(ctx context.Context, id, user string) error
	RemoveLike(ctx context.Context, id, user string) error
	IsLikedByUser(ctx context.Context, id, userID string) (bool, error)
}

type postRepo struct {
	mongoClient *mongo.Client
	dbName      string
	logger      *zap.Logger
}

func NewPostRepository(client *mongo.Client, dbName string, logger *zap.Logger) PostRepository {
	repo := &postRepo{
		mongoClient: client,
		dbName:      dbName,
		logger:      logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.ensureIndexes(ctx); err != nil {
		logger.Fatal("failed to create indexes", zap.Error(err))
	}

	return repo
}

func (r *postRepo) PostCollection() *mongo.Collection {
	return r.mongoClient.Database(r.dbName).Collection("posts")
}

func (r *postRepo) likesCollection() *mongo.Collection {
	return r.mongoClient.Database(r.dbName).Collection("post_likes")
}

func (r *postRepo) ensureIndexes(ctx context.Context) error {

	postIndexes := []mongo.IndexModel{
		{
			Keys:    bson.M{"slug": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{"tags": 1},
		},
		{
			Keys: bson.M{"topic": 1},
		},
		{
			Keys: bson.M{"likes_count": -1},
		},
		{
			Keys: bson.M{"created_at": -1},
		},
	}

	likesIndexes := []mongo.IndexModel{
		{
			Keys: bson.M{
				"post_id": 1,
				"user_id": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := r.PostCollection().Indexes().CreateMany(ctx, postIndexes)
	_, err = r.likesCollection().Indexes().CreateMany(ctx, likesIndexes)

	return err
}

func (r *postRepo) Create(ctx context.Context, post *model.Post) error {
	post.ID = primitive.NewObjectID()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	_, err := r.PostCollection().InsertOne(ctx, post)
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
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	var post model.Post

	// 3️⃣ Поиск
	err = r.PostCollection().FindOne(ctx, filter).Decode(&post)
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

func (r *postRepo) GetBySlug(ctx context.Context, slug string) (*model.Post, error) {

	filter := bson.M{
		"slug":       slug,
		"deleted_at": bson.M{"$eq": nil},
	}

	var post model.Post

	err := r.PostCollection().FindOne(ctx, filter).Decode(&post)
	if err != nil {

		if err == mongo.ErrNoDocuments {
			r.logger.Warn("post not found",
				zap.String("slug", slug),
			)
			return nil, ErrNotFound
		}

		r.logger.Error("failed to get post by slug",
			zap.Error(err),
			zap.String("slug", slug),
		)
		return nil, err
	}

	return &post, nil
}

func (r *postRepo) Update(ctx context.Context, post *model.Post) error {

	if post.ID.IsZero() {
		return ErrNotFound
	}

	post.UpdatedAt = time.Now()

	filter := bson.M{
		"_id":        post.ID,
		"deleted_at": bson.M{"$eq": nil},
	}

	update := bson.M{
		"$set": bson.M{
			"title":          post.Title,
			"content":        post.Content,
			"topic":          post.Topic,
			"tags":           post.Tags,
			"likes_count":    post.LikesCount,
			"comments_count": post.CommentsCount,
			"slug":           post.Slug,
			"updated_at":     post.UpdatedAt,
		},
	}

	result, err := r.PostCollection().UpdateOne(ctx, filter, update)
	if err != nil {

		if mongo.IsDuplicateKeyError(err) {
			r.logger.Warn("slug already exists",
				zap.String("slug", post.Slug),
			)
			return ErrSlugExists
		}

		r.logger.Error("failed to update post",
			zap.Error(err),
			zap.String("post_id", post.ID.Hex()),
		)

		return err
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("post not found for update",
			zap.String("post_id", post.ID.Hex()),
		)
		return ErrNotFound
	}

	r.logger.Info("post updated",
		zap.String("post_id", post.ID.Hex()),
	)

	return nil
}

func (r *postRepo) MarkAsDeleted(ctx context.Context, id string) error {

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
			"status":     "deleted",
			"updated_at": time.Now(),
		},
	}

	result, err := r.PostCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to soft delete post",
			zap.Error(err),
			zap.String("post_id", id),
		)
		return err
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("post not found or already deleted",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	r.logger.Info("post soft deleted",
		zap.String("post_id", id),
	)

	return nil
}

func (r *postRepo) Delete(ctx context.Context, id string) error {

	// 1️⃣ Конвертация ID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	filter := bson.M{"_id": objectID}

	// 2️⃣ Удаление
	result, err := r.PostCollection().DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Error("failed to delete post",
			zap.Error(err),
			zap.String("post_id", id),
		)
		return err
	}

	// 3️⃣ Проверка найден ли документ
	if result.DeletedCount == 0 {
		r.logger.Warn("post not found for delete",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	//TODO Удаляем все лайки этого поста
	_, err = r.likesCollection().DeleteMany(ctx, bson.M{"post_id": objectID})
	if err != nil {
		r.logger.Error("failed to delete post likes",
			zap.Error(err),
			zap.String("post_id", id),
		)
		return err
	}

	r.logger.Info("post deleted",
		zap.String("post_id", id),
	)

	return nil
}

func (r *postRepo) IncrementViews(ctx context.Context, id string) error {

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	update := bson.M{
		"$inc": bson.M{
			"views": 1,
		},
	}

	result, err := r.PostCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to increment views",
			zap.Error(err),
			zap.String("post_id", objectID.Hex()),
		)
		return err
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("post not found or already deleted",
			zap.String("post_id", id),
		)
		return ErrNotFound
	}

	return nil
}

func (r *postRepo) AddLike(ctx context.Context, id, userID string) error {

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}

	postLike := &model.PostLike{
		ID:        primitive.NewObjectID(),
		PostID:    objectID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	// 1️⃣ вставляем лайк
	_, err = r.likesCollection().InsertOne(ctx, postLike)
	if err != nil {

		if mongo.IsDuplicateKeyError(err) {
			// пользователь уже лайкнул — просто выходим
			return nil
		}

		return err
	}

	// 2️⃣ пробуем инкремент
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	update := bson.M{
		"$inc": bson.M{
			"likes_count": 1,
		},
	}

	_, err = r.PostCollection().UpdateOne(ctx, filter, update)
	if err == nil {
		return nil
	}

	// 3️⃣ если инкремент не удался — пересчитываем
	r.logger.Warn("increment failed, recalculating likes_count",
		zap.String("post_id", id),
		zap.Error(err),
	)

	return r.recalculateLikesCount(ctx, objectID)
}

func (r *postRepo) RemoveLike(ctx context.Context, id, likeID string) error {
	postObjectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}

	likeObjectID, err := primitive.ObjectIDFromHex(likeID)
	if err != nil {
		return ErrNotFound
	}

	likesFilter := bson.M{
		"_id":     likeObjectID,
		"post_id": postObjectID,
	}

	result, err := r.likesCollection().DeleteOne(ctx, likesFilter)
	if err != nil {
		r.logger.Error("failed to delete like",
			zap.Error(err),
			zap.String("like_id", likeID),
		)
		return err
	}

	if result.DeletedCount == 0 {
		r.logger.Warn("like not found for delete",
			zap.String("like_id", likeID),
		)
		return ErrNotFound
	}

	r.logger.Info("like deleted",
		zap.String("like_id", likeID),
	)

	postFilter := bson.M{
		"_id":        postObjectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	update := bson.M{
		"$inc": bson.M{
			"likes_count": -1,
		},
	}

	_, err = r.PostCollection().UpdateOne(ctx, postFilter, update)
	if err == nil {
		return nil
	}

	r.logger.Warn("decrement failed, recalculating likes_count",
		zap.String("post_id", id),
		zap.Error(err),
	)

	return r.recalculateLikesCount(ctx, postObjectID)
}

func (r *postRepo) recalculateLikesCount(ctx context.Context, postID primitive.ObjectID) error {

	count, err := r.likesCollection().CountDocuments(ctx, bson.M{
		"post_id": postID,
	})
	if err != nil {
		return err
	}

	_, err = r.PostCollection().UpdateOne(ctx,
		bson.M{"_id": postID},
		bson.M{"$set": bson.M{"likes_count": count}},
	)

	return err
}

func (r *postRepo) IsLikedByUser(ctx context.Context, postID, userID string) (bool, error) {

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return false, ErrNotFound
	}

	filter := bson.M{
		"post_id": objectID,
		"user_id": userID,
	}

	err = r.likesCollection().FindOne(ctx, filter).Err()

	if err == mongo.ErrNoDocuments {
		return false, nil
	}

	if err != nil {
		r.logger.Error("failed to check like existence",
			zap.Error(err),
		)
		return false, err
	}

	return true, nil
}

// TODO delete
func (r *postRepo) GetPostWithLikeState(
	ctx context.Context,
	postID string,
	userID string,
) (*model.PostWithLikeState, error) {

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, ErrNotFound
	}

	// 1️⃣ получаем пост
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$eq": nil},
	}

	var post model.Post

	err = r.PostCollection().FindOne(ctx, filter).Decode(&post)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 2️⃣ проверяем лайк
	likeFilter := bson.M{
		"post_id": objectID,
		"user_id": userID,
	}

	err = r.likesCollection().FindOne(ctx, likeFilter).Err()

	isLiked := true

	if err == mongo.ErrNoDocuments {
		isLiked = false
	} else if err != nil {
		return nil, err
	}

	return &model.PostWithLikeState{
		Post:    &post,
		IsLiked: isLiked,
	}, nil
}

func (r *postRepo) ListPostsAdvanced(
	ctx context.Context,
	userID string,
	topic string,
	tag string,
	sortBy string,
	sortOrder int,
	page, limit int64,
) (*model.PaginatedPostsWithLikeState, error) {

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	if page <= 0 {
		page = 1
	}

	if sortOrder != 1 && sortOrder != -1 {
		sortOrder = -1
	}

	// 🔥 1️⃣ динамический фильтр
	filter := bson.M{
		"deleted_at": bson.M{"$eq": nil},
	}

	if topic != "" {
		filter["topic"] = topic
	}

	if tag != "" {
		filter["tags"] = tag
	}

	// 🔥 2️⃣ определяем поле сортировки
	sortField := "created_at"

	if sortBy == "likes" {
		sortField = "likes_count"
	}

	// 🔥 3️⃣ считаем total
	total, err := r.PostCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	skip := (page - 1) * limit

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.PostCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []*model.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return &model.PaginatedPostsWithLikeState{
			Items: []*model.PostWithLikeState{},
			Total: total,
			Page:  page,
			Limit: limit,
		}, nil
	}

	// 🔥 4️⃣ собираем postIDs
	postIDs := make([]primitive.ObjectID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}

	// 🔥 5️⃣ получаем лайки пользователя
	likesFilter := bson.M{
		"user_id": userID,
		"post_id": bson.M{"$in": postIDs},
	}

	likesCursor, err := r.likesCollection().Find(ctx, likesFilter)
	if err != nil {
		return nil, err
	}
	defer likesCursor.Close(ctx)

	likedMap := make(map[primitive.ObjectID]bool)

	for likesCursor.Next(ctx) {
		var like model.PostLike
		if err := likesCursor.Decode(&like); err != nil {
			return nil, err
		}
		likedMap[like.PostID] = true
	}

	// 🔥 6️⃣ собираем результат
	result := make([]*model.PostWithLikeState, 0, len(posts))

	for _, post := range posts {
		result = append(result, &model.PostWithLikeState{
			Post:    post,
			IsLiked: likedMap[post.ID],
		})
	}

	return &model.PaginatedPostsWithLikeState{
		Items: result,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
