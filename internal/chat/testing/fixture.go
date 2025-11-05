package testing

import (
	"context"
	"testing"
	"time"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Fixture struct {
	*model.Repository
	test   *testing.T
	defers []func()
}

func WithFixture(runner func(t *testing.T, f *Fixture)) func(t *testing.T) {
	return func(t *testing.T) {
		f := &Fixture{Repository: model.New(ConnectMongo()), test: t}
		defer f.Teardown()
		runner(t, f)
	}
}

func (f *Fixture) CreateConversation(mods ...func(*model.Conversation)) *model.Conversation {
	c := &model.Conversation{
		ID:        primitive.NewObjectID(),
		Title:     uuid.New().String(),
		CreatedAt: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
		Messages: []*model.Message{{
			ID:        primitive.NewObjectID(),
			Role:      model.RoleUser,
			Content:   "What is the weather like today?",
			CreatedAt: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
		}},
	}

	for _, mod := range mods {
		mod(c)
	}

	ctx := context.Background()

	if err := f.Repository.CreateConversation(ctx, c); err != nil {
		f.test.Fatalf("failed to create conversation: %v", err)
	}

	f.defers = append(f.defers, func() {
		if err := f.Repository.DeleteConversation(ctx, c.ID.Hex()); err != nil {
			f.test.Logf("failed to cleanup conversation %s: %v", c.ID.Hex(), err)
		}
	})

	return c
}

func (f *Fixture) Teardown() {
	for _, d := range f.defers {
		d()
	}
}
