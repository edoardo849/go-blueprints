package main

import (
	"flag"
	"log"
	"net/http"

	"context"

	"gopkg.in/mgo.v2"
)

func main() {

	var (
		addr  = flag.String("addr", ":8080", "endpoint address")
		mongo = flag.String("mongo", "192.168.99.100", "mongodb address")
	)
	log.Println("Dialing mongo", *mongo)
	db, err := mgo.Dial(*mongo)
	if err != nil {
		log.Fatalln("failed to connect to mongo:", err)
	}
	defer db.Close()
	s := &Server{
		db: db,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/polls/", withCORS(withAPIKey(s.handlePolls)))
	log.Println("Starting web server on", *addr)
	http.ListenAndServe(":8080", mux)
	log.Println("Stopping...")
}

// Server is the API server.
type Server struct {
	db *mgo.Session
}

func withCORS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Location")
		fn(w, r)
	}
}

// this pattern https://medium.com/@matryer/context-keys-in-go-5312346a868d#.lrgjb2hj6
// is for sharing state only.
type contextKey struct {
	name string
}

var contextKeyAPIKey = &contextKey{"api-key"}

// APIKey is a helper function that is getting the value
// for the user
// An alternative patter, is to keep completely private the api key
// by returning the resource. In this example, a User
//```go
// func User(ctx context.Context) (*User, error) {
//    tok, ok := authTokenFromContext(ctx)
//    if !ok {
//        return nil, ErrNoUser
//    }
//    user, err := LookupUserByToken(ctx, tok)
//    if err != nil {
//        return nil, err
//    }
//    return user, nil
//}
//```
func APIKey(ctx context.Context) (string, bool) {
	key := ctx.Value(contextKeyAPIKey)
	if key == nil {
		return "", false
	}
	keystr, ok := key.(string)
	return keystr, ok
}

func withAPIKey(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		key := r.URL.Query().Get("key")
		if !isValidAPIKey(key) {
			respondErr(w, r, http.StatusUnauthorized, "invalid API key")
			return
		}
		// through the decorator pattern, we are adding context to our
		// handlers.
		ctx := context.WithValue(r.Context(), contextKeyAPIKey, key)
		fn(w, r.WithContext(ctx))
	}
}

func isValidAPIKey(key string) bool {
	return key == "abc123"
}
