package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"noxis/internal/entity"
)

const connectTimeout = 10 * time.Second

// connect builds the URI from the structured fields and opens a client. The database doubles as the auth source unless params say otherwise, which is what an operator expects from "user, password, database".
func connect(ctx context.Context, c entity.Connection) (*mongo.Client, error) {
	uri, err := dsn(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", c.Name, err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("connect %s: %w", c.Name, err)
	}
	return client, nil
}

// dsn builds a mongodb:// URI.
func dsn(c entity.Connection) (string, error) {
	host := c.Host
	if c.Port > 0 {
		host = c.Host + ":" + strconv.Itoa(c.Port)
	}
	u := &url.URL{Scheme: "mongodb", Host: host}
	if c.User != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		} else {
			u.User = url.User(c.User)
		}
	}
	q := u.Query()
	if c.Database != "" {
		u.Path = "/" + c.Database
		q.Set("authSource", c.Database)
	}
	for key, value := range c.Params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	if strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("connection %q has no host", c.Name)
	}
	return u.String(), nil
}
