package model

import (
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Conversation struct {
	ID        primitive.ObjectID `bson:"_id"`
	Title     string             `bson:"subject"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Messages  []*Message         `bson:"messages"`
}

func (c *Conversation) Proto() *pb.Conversation {
	proto := &pb.Conversation{
		Id:        c.ID.Hex(),
		Title:     c.Title,
		Timestamp: timestamppb.New(c.UpdatedAt),
	}

	for _, m := range c.Messages {
		proto.Messages = append(proto.Messages, m.Proto())
	}

	return proto
}
