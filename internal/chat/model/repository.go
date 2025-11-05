package model

import (
	"context"
	"errors"

	"github.com/twitchtv/twirp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	conversationCollection = "conversations"
)

type Repository struct {
	conn *mongo.Database
}

func New(conn *mongo.Database) *Repository {
	return &Repository{
		conn: conn,
	}
}

func (r *Repository) CreateConversation(ctx context.Context, c *Conversation) error {
	_, err := r.conn.Collection(conversationCollection).InsertOne(ctx, c)
	return err
}

func (r *Repository) DescribeConversation(ctx context.Context, id string) (*Conversation, error) {
	var c Conversation

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, twirp.NotFoundError("invalid conversation ID")
	}

	err = r.conn.Collection(conversationCollection).FindOne(ctx, map[string]any{"_id": oid}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, twirp.NotFoundError("conversation not found")
	}

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *Repository) ListConversations(ctx context.Context) ([]*Conversation, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.conn.Collection(conversationCollection).
		Find(ctx, map[string]any{}, opts)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = cursor.Close(ctx)
	}()

	var items []*Conversation

	for cursor.Next(ctx) {
		var c Conversation

		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}

		items = append(items, &c)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) UpdateConversation(ctx context.Context, c *Conversation) error {
	_, err := r.conn.Collection(conversationCollection).UpdateOne(ctx,
		map[string]any{"_id": c.ID},
		map[string]any{"$set": c})

	if errors.Is(err, mongo.ErrNoDocuments) {
		return twirp.NotFoundError("conversation not found")
	}

	return err
}

func (r *Repository) DeleteConversation(ctx context.Context, id string) error {
	_, err := r.conn.Collection(conversationCollection).DeleteOne(ctx, map[string]any{"_id": id})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return twirp.NotFoundError("conversation not found")
	}

	return err
}
