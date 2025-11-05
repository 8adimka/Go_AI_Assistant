package mongox

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MustConnect(uri, dbname string) *mongo.Database {
	client, err := mongo.Connect(context.Background(), options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
		SetBSONOptions(&options.BSONOptions{NilSliceAsEmpty: true}))

	if err != nil {
		panic(err)
	}

	return client.Database(dbname)
}
