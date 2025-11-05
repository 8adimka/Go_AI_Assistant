package model

import "github.com/acai-travel/tech-challenge/internal/pb"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

func (r Role) Proto() pb.Conversation_Role {
	switch r {
	case RoleUser:
		return pb.Conversation_USER
	case RoleAssistant:
		return pb.Conversation_ASSISTANT
	default:
		return 0
	}
}
