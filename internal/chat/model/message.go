package model

import (
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Message struct {
	ID        primitive.ObjectID `bson:"_id"`
	Role      Role               `bson:"role"`
	Content   string             `bson:"content"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

func (m *Message) Proto() *pb.Conversation_Message {
	return &pb.Conversation_Message{
		Id:        m.ID.Hex(),
		Role:      m.Role.Proto(),
		Content:   m.Content,
		Timestamp: timestamppb.New(m.CreatedAt),
	}
}
