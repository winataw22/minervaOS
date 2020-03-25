package mw

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

type (
	dbMiddlewareKey struct{}
)

// DatabaseMiddleware middleware
type DatabaseMiddleware struct {
	name   string
	client *mongo.Client
}

// NewDatabaseMiddleware creates a new database middleware
func NewDatabaseMiddleware(dbName string, client *mongo.Client) (*DatabaseMiddleware, error) {

	return &DatabaseMiddleware{name: dbName, client: client}, nil
}

// Middleware is the middleware function
func (d *DatabaseMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), dbMiddlewareKey{}, d.client.Database(d.name))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Database return database as configured in the middleware
func (d *DatabaseMiddleware) Database() *mongo.Database {
	return d.client.Database(d.name)
}

// Database gets the database configured on the request
func Database(r *http.Request) *mongo.Database {
	v := r.Context().Value(dbMiddlewareKey{})
	if v == nil {
		panic("DatabaseMiddleware is not configured")
	}

	return v.(*mongo.Database)
}
