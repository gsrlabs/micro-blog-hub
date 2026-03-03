package repository

import (
	"context"
	"time"

	"github.com/gsrlabs/micro-blog-hub/post-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *model.Comment) error
	UpdateComment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, commentID string) error
	ListCommentsByPost(ctx context.Context, postID string, page, limit int64) (*model.ListCommentsResult, error)
	AddLike(ctx context.Context, commentID, userID string) error
	RemoveLike(ctx context.Context, commentID, likeID string) error
	ListCommentsAdvanced(
		ctx context.Context,
		postID string,
		userID string,
		page, limit int64,
		sortDesc bool,
	) ([]*model.CommentWithLikeState, int64, error)
}

type commentRepo struct {
	mongoClient *mongo.Client
	dbName      string
	logger      *zap.Logger
}

func NewCommentRepository(client *mongo.Client, dbName string, logger *zap.Logger) CommentRepository {
	return &commentRepo{
		mongoClient: client,
		dbName:      dbName,
		logger:      logger,
	}
}

func (r *commentRepo) commentsCollection() *mongo.Collection {
	return r.mongoClient.Database(r.dbName).Collection("comments")
}

func (r *commentRepo) commentsLikeCollection() *mongo.Collection {
	return r.mongoClient.Database(r.dbName).Collection("comments_likes")
}

func (r *commentRepo) ensureIndexes(ctx context.Context) error {

	comments := []mongo.IndexModel{
		{
			Keys: bson.M{"post_id": 1},
		},
		{
			Keys: bson.M{"created_at": -1},
		},
	}

	commentsLike := []mongo.IndexModel{
		{
			Keys: bson.M{
				"comment_id": 1,
				"user_id":    1,
			},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := r.commentsCollection().Indexes().CreateMany(ctx, comments)
	_, err = r.commentsLikeCollection().Indexes().CreateMany(ctx, commentsLike)
	return err
}

func (r *commentRepo) CreateComment(ctx context.Context, comment *model.Comment) error {
	collection := r.commentsCollection()

	// 🔹 1. Заполняем системные поля
	comment.ID = primitive.NewObjectID()
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	comment.LikesCount = 0

	// 🔹 2. Вставляем в MongoDB
	_, err := collection.InsertOne(ctx, comment)
	if err != nil {
		r.logger.Error("failed to insert comment",
			zap.Error(err),
			zap.String("post_id", comment.PostID.Hex()),
			zap.String("author_id", comment.AuthorID),
		)
		return err
	}

	r.logger.Info("comment created",
		zap.String("comment_id", comment.ID.Hex()),
		zap.String("post_id", comment.PostID.Hex()),
	)

	return nil
}

func (r *commentRepo) UpdateComment(ctx context.Context, comment *model.Comment) error {
	if comment.ID.IsZero() {
		r.logger.Warn("update called with empty comment ID")
		return ErrNotFound
	}

	comment.UpdatedAt = time.Now()

	filter := bson.M{
		"_id": comment.ID,
	}

	update := bson.M{
		"$set": bson.M{
			"content":     comment.Content,
			"likes_count": comment.LikesCount,
			"updated_at":  comment.UpdatedAt,
		},
	}

	result, err := r.commentsCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to update comment",
			zap.Error(err),
			zap.String("comment_id", comment.ID.Hex()),
		)
		return err
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("comment not found for update",
			zap.String("comment_id", comment.ID.Hex()),
		)
		return ErrNotFound
	}

	r.logger.Info("comment updated",
		zap.String("comment_id", comment.ID.Hex()),
	)

	return nil
}

func (r *commentRepo) DeleteComment(ctx context.Context, commentID string) error {
	// 1️⃣ Преобразуем ID в ObjectID
	objID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		r.logger.Warn("invalid comment id format",
			zap.String("comment_id", commentID),
		)
		return ErrNotFound
	}

	// 2️⃣ Удаляем сам комментарий
	filter := bson.M{"_id": objID}
	result, err := r.commentsCollection().DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Error("failed to delete comment",
			zap.Error(err),
			zap.String("comment_id", commentID),
		)
		return err
	}

	if result.DeletedCount == 0 {
		r.logger.Warn("comment not found for delete",
			zap.String("comment_id", commentID),
		)
		return ErrNotFound
	}

	r.logger.Info("comment deleted",
		zap.String("comment_id", commentID),
	)

	// 3️⃣ Удаляем все лайки этого комментария
	_, err = r.commentsLikeCollection().DeleteMany(ctx, bson.M{"comment_id": objID})
	if err != nil {
		r.logger.Error("failed to delete comment likes",
			zap.Error(err),
			zap.String("comment_id", commentID),
		)
		return err
	}

	return nil
}

func (r *commentRepo) ListCommentsByPost(
	ctx context.Context,
	postID string,
	page, limit int64,
) (*model.ListCommentsResult, error) {

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	skip := (page - 1) * limit

	postObjectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", postID),
		)
		return nil, ErrNotFound
	}

	filter := bson.M{
		"post_id": postObjectID,
	}

	// 🔹 Считаем общее количество комментариев
	totalCount, err := r.commentsCollection().CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to count comments",
			zap.Error(err),
			zap.String("post_id", postID),
		)
		return nil, err
	}

	// 🔹 Получаем сами комментарии
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}). // по возрастанию
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.commentsCollection().Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("failed to list comments",
			zap.Error(err),
			zap.String("post_id", postID),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []*model.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	r.logger.Info("comments fetched",
		zap.String("post_id", postID),
		zap.Int("count", len(comments)),
		zap.Int64("total_count", totalCount),
	)

	return &model.ListCommentsResult{
		Comments:   comments,
		TotalCount: totalCount,
	}, nil
}

func (r *commentRepo) AddLike(ctx context.Context, commentID, userID string) error {
	// 1️⃣ Преобразуем ID
	cmtObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		r.logger.Warn("invalid comment id format",
			zap.String("comment_id", commentID),
		)
		return ErrNotFound
	}

	// 2️⃣ Создаём объект лайка
	like := &model.CommentLike{
		ID:        primitive.NewObjectID(),
		CommentID: cmtObjID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	// 3️⃣ Вставляем лайк
	_, err = r.commentsLikeCollection().InsertOne(ctx, like)
	if err != nil {
		r.logger.Error("failed to insert comment like",
			zap.Error(err),
			zap.String("comment_id", commentID),
			zap.String("user_id", userID),
		)
		return err
	}

	// 4️⃣ Увеличиваем счетчик лайков у комментария
	filter := bson.M{"_id": cmtObjID}
	update := bson.M{"$inc": bson.M{"likes_count": 1}}

	_, err = r.commentsCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Warn("failed to increment likes_count, recalculating",
			zap.String("comment_id", commentID),
			zap.Error(err),
		)

		// При неудаче можно пересчитать количество лайков заново
		count, _ := r.commentsLikeCollection().CountDocuments(ctx, bson.M{"post_id": cmtObjID})
		r.commentsCollection().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"likes_count": count}})
	}

	r.logger.Info("like added to comment",
		zap.String("comment_id", commentID),
		zap.String("user_id", userID),
	)

	return nil
}

func (r *commentRepo) RemoveLike(ctx context.Context, commentID, likeID string) error {
	cmtObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return ErrNotFound
	}

	likeObjID, err := primitive.ObjectIDFromHex(likeID)
	if err != nil {
		return ErrNotFound
	}

	// 1️⃣ Удаляем лайк
	result, err := r.commentsLikeCollection().DeleteOne(ctx, bson.M{"_id": likeObjID})
	if err != nil {
		r.logger.Error("failed to delete comment like",
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

	// 2️⃣ Декрементируем счетчик лайков
	filter := bson.M{"_id": cmtObjID}
	update := bson.M{"$inc": bson.M{"likes_count": -1}}

	_, err = r.commentsCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Warn("failed to decrement likes_count, recalculating",
			zap.String("comment_id", commentID),
			zap.Error(err),
		)

		// Пересчитываем количество лайков заново
		count, _ := r.commentsLikeCollection().CountDocuments(ctx, bson.M{"post_id": cmtObjID})
		r.commentsCollection().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"likes_count": count}})
	}

	r.logger.Info("like removed from comment",
		zap.String("comment_id", commentID),
		zap.String("like_id", likeID),
	)

	return nil
}

func (r *commentRepo) IsLikedByUser(ctx context.Context, commentID, userID string) (bool, error) {
	// 1️⃣ Преобразуем commentID в ObjectID
	cmtObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		r.logger.Warn("invalid comment id format",
			zap.String("comment_id", commentID),
		)
		return false, ErrNotFound
	}

	// 2️⃣ Фильтр: ищем лайк пользователя для данного комментария
	filter := bson.M{
		"post_id": cmtObjID,
		"user_id": userID,
	}

	// 3️⃣ CountDocuments — проверяем, есть ли такой лайк
	count, err := r.commentsLikeCollection().CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to check like for comment",
			zap.String("comment_id", commentID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return false, err
	}

	// 4️⃣ Если count > 0 — значит лайк есть
	return count > 0, nil
}

func (r *commentRepo) AddLikeToComment(ctx context.Context, commentID, userID string) error {
	commentObjectID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		r.logger.Warn("invalid comment id format",
			zap.String("comment_id", commentID),
		)
		return ErrNotFound
	}

	// Создаем запись лайка
	commentLike := &model.CommentLike{
		ID:        primitive.NewObjectID(),
		CommentID: commentObjectID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	// Вставляем лайк в коллекцию comment_likes
	_, err = r.commentsLikeCollection().InsertOne(ctx, commentLike)
	if err != nil {
		// Проверка на дубликат (уникальный индекс)
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Warn("like already exists",
				zap.String("comment_id", commentID),
				zap.String("user_id", userID),
			)
			return nil
		}

		r.logger.Error("failed to insert comment like",
			zap.Error(err),
			zap.String("comment_id", commentID),
			zap.String("user_id", userID),
		)
		return err
	}

	// Инкрементируем счетчик лайков у комментария
	filter := bson.M{"_id": commentObjectID}
	update := bson.M{"$inc": bson.M{"likes_count": 1}}

	result, err := r.commentsCollection().UpdateOne(ctx, filter, update)
	if err != nil || result.MatchedCount == 0 {
		r.logger.Warn("failed to increment likes_count or comment not found",
			zap.String("comment_id", commentID),
			zap.Error(err),
		)

		// Если инкремент не удался, пересчитываем
		return r.recalculateCommentLikesCount(ctx, commentObjectID)
	}

	r.logger.Info("comment like added",
		zap.String("comment_id", commentID),
		zap.String("user_id", userID),
	)

	return nil
}

func (r *commentRepo) RemoveLikeFromComment(ctx context.Context, commentID, likeID string) error {
	commentObjectID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return ErrNotFound
	}

	likeObjectID, err := primitive.ObjectIDFromHex(likeID)
	if err != nil {
		return ErrNotFound
	}

	// Удаляем лайк
	result, err := r.commentsLikeCollection().DeleteOne(ctx, bson.M{"_id": likeObjectID})
	if err != nil {
		r.logger.Error("failed to delete comment like",
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

	// Декрементируем счетчик лайков комментария
	filter := bson.M{"_id": commentObjectID}
	update := bson.M{"$inc": bson.M{"likes_count": -1}}

	_, err = r.commentsCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Warn("decrement failed, recalculating likes_count",
			zap.String("comment_id", commentID),
			zap.Error(err),
		)

		// Пересчет на случай ошибок
		return r.recalculateCommentLikesCount(ctx, commentObjectID)
	}

	r.logger.Info("comment like removed",
		zap.String("comment_id", commentID),
		zap.String("like_id", likeID),
	)

	return nil
}

func (r *commentRepo) recalculateCommentLikesCount(ctx context.Context, commentID primitive.ObjectID) error {

	count, err := r.commentsLikeCollection().CountDocuments(ctx, bson.M{
		"comment_id": commentID,
	})
	if err != nil {
		return err
	}

	_, err = r.commentsCollection().UpdateOne(ctx,
		bson.M{"_id": commentID},
		bson.M{"$set": bson.M{"likes_count": count}},
	)

	return err
}

func (r *commentRepo) ListCommentsAdvanced(
	ctx context.Context,
	postID string,
	userID string,
	page, limit int64,
	sortDesc bool, // true = newest first, false = oldest first
) ([]*model.CommentWithLikeState, int64, error) {

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	skip := (page - 1) * limit

	// 1️⃣ Преобразуем postID
	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		r.logger.Warn("invalid post id format",
			zap.String("post_id", postID),
		)
		return nil, 0, ErrNotFound
	}

	// 2️⃣ Фильтр комментариев по postID
	filter := bson.M{"post_id": postObjID}

	// 🔹 Получаем total count
	totalCount, err := r.commentsCollection().CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to count comments",
			zap.Error(err),
			zap.String("post_id", postID),
		)
		return nil, 0, err
	}

	// 3️⃣ Опции поиска с пагинацией и сортировкой
	sortOrder := 1
	if sortDesc {
		sortOrder = -1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: sortOrder}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.commentsCollection().Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("failed to list comments",
			zap.Error(err),
			zap.String("post_id", postID),
		)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*model.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	if len(comments) == 0 {
		return []*model.CommentWithLikeState{}, totalCount, nil
	}

	// 4️⃣ Получаем все лайки пользователя одним запросом
	commentIDs := make([]primitive.ObjectID, 0, len(comments))
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
	}

	likesFilter := bson.M{
		"user_id":    userID,
		"comment_id": bson.M{"$in": commentIDs},
	}

	likesCursor, err := r.commentsLikeCollection().Find(ctx, likesFilter)
	if err != nil {
		return nil, 0, err
	}
	defer likesCursor.Close(ctx)

	likedMap := make(map[primitive.ObjectID]bool)
	for likesCursor.Next(ctx) {
		var like model.CommentLike
		if err := likesCursor.Decode(&like); err != nil {
			return nil, 0, err
		}
		likedMap[like.CommentID] = true
	}

	// 5️⃣ Собираем результат
	result := make([]*model.CommentWithLikeState, 0, len(comments))
	for _, c := range comments {
		result = append(result, &model.CommentWithLikeState{
			Comment: c,
			IsLiked: likedMap[c.ID],
		})
	}

	return result, totalCount, nil
}
