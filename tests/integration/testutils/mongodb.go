package testutils

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBContainer represents a test MongoDB container
type MongoDBContainer struct {
	testcontainers.Container
	URI string
}

// SetupMongoDBContainer creates and starts a MongoDB container for testing
func SetupMongoDBContainer(ctx context.Context) (*MongoDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:7.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(30 * time.Second),
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "test",
			"MONGO_INITDB_ROOT_PASSWORD": "test",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %v", err)
	}

	uri := fmt.Sprintf("mongodb://test:test@%s:%s", host, port.Port())

	return &MongoDBContainer{
		Container: container,
		URI:       uri,
	}, nil
}

// ConnectMongoDB connects to the MongoDB container
func ConnectMongoDB(ctx context.Context, uri string) (*mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
		SetBSONOptions(&options.BSONOptions{NilSliceAsEmpty: true}))

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	dbName := "test_" + time.Now().Format("20060102150405")
	return client.Database(dbName), nil
}

// WithMongoDBContainer is a test helper that provides a MongoDB container
func WithMongoDBContainer(t *testing.T, testFunc func(ctx context.Context, db *mongo.Database)) {
	ctx := context.Background()

	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" && os.Getenv("DOCKER_HOST") == "" {
		t.Skip("Skipping test that requires Docker in CI environment")
	}

	container, err := SetupMongoDBContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup MongoDB container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}()

	db, err := ConnectMongoDB(ctx, container.URI)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Run the test function
	testFunc(ctx, db)

	// Cleanup test database
	if err := db.Drop(ctx); err != nil {
		t.Logf("Failed to drop test database: %v", err)
	}
}
