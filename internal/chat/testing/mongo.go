package testing

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database
var once sync.Once

func ConnectMongo() *mongo.Database {
	once.Do(func() {
		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			uri = "mongodb://acai:travel@localhost:27017"
		}

		dbname := os.Getenv("MONGODB_DATABASE")
		if dbname == "" {
			dbname = "acai_testing"
		}

		client, err := mongo.Connect(context.Background(), options.Client().
			ApplyURI(uri).
			SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
			SetBSONOptions(&options.BSONOptions{NilSliceAsEmpty: true}))

		if err != nil {
			panic(fmt.Errorf("failed to connect to MongoDB: %v", err))
		}

		db = client.Database(dbname)
	})

	return db
}
