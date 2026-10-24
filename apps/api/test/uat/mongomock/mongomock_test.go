package mongomock

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func connectMock(t *testing.T, server *Server) *mongo.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://mock:mock@127.0.0.1:%d/?authSource=admin", server.Port())).SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return client
}

func TestServesFind(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Answer(bson.M{"sku": "a"}, bson.M{"sku": "b"})
	client := connectMock(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := client.Database("demo").Collection("items").Find(ctx, bson.M{"sku": "a"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		t.Fatalf("cursor: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs = %v, want 2", docs)
	}
	if filters := server.Filters(); len(filters) == 0 || filters[0]["sku"] != "a" {
		t.Fatalf("filters = %v", filters)
	}
}

func TestServesError(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Fail("collection missing does not exist")
	client := connectMock(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Database("demo").Collection("missing").FindOne(ctx, bson.M{}).Err(); err == nil {
		t.Fatal("expected a target error")
	}
}
