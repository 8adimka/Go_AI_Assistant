package mongox

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MustConnect() *mongo.Database {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://acai:travel@localhost:27017"
	}

	dbname := os.Getenv("MONGODB_DATABASE")
	if dbname == "" {
		dbname = "acai"
	}

	client, err := mongo.Connect(context.Background(), options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
		SetBSONOptions(&options.BSONOptions{NilSliceAsEmpty: true}))

	if err != nil {
		panic(err)
	}

	return client.Database(dbname)
}
