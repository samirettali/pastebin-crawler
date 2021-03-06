package storage

import (
	"container/list"
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	pb "github.com/samirettali/go-pastebin"
)

type MongoConfig struct {
	URI        string `required:"true"`
	Database   string `required:"true"`
	Collection string `required:"true"`
}

// MongoStorage is an implementation of the Storage interface
type MongoStorage struct {
	Config *MongoConfig

	col   *mongo.Collection
	mutex sync.Mutex
	cache *list.List
}

// Init initializes the collection pointer
func (s *MongoStorage) Init() error {
	var err error
	client, err := mongo.NewClient(options.Client().ApplyURI(s.Config.URI))
	if err != nil {
		return fmt.Errorf("Could not create the client: %w", err)
	}
	if err = client.Connect(context.Background()); err != nil {
		return fmt.Errorf("Could not connect to the DB: %w", err)
	}
	db := client.Database(s.Config.Database)
	s.col = db.Collection(s.Config.Collection)
	s.cache = list.New()
	return nil
}

// IsSaved checks if the paste is already saved
func (s *MongoStorage) IsSaved(key string) (bool, error) {
	if s.isInCache(key) {
		return true, nil
	}

	paste := pb.Paste{}
	filter := &bson.M{
		"key": key,
	}

	err := s.col.FindOne(context.Background(), filter).Decode(&paste)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, fmt.Errorf("Could not search for paste: %w", err)
	}

	return true, nil
}

// Save saves a paste
func (s *MongoStorage) Save(paste pb.Paste) error {
	_, err := s.col.InsertOne(context.Background(), paste)
	if err != nil {
		return fmt.Errorf("Could not save the paste: %w", err)
	}
	s.addToCache(paste.Key)
	return nil
}

func (s *MongoStorage) addToCache(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cache.PushBack(key)
	if s.cache.Len() > 250 {
		e := s.cache.Front()
		s.cache.Remove(e)
	}
}

func (s *MongoStorage) isInCache(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for c := s.cache.Front(); c != nil; c = c.Next() {
		if c.Value == key {
			return true
		}
	}
	return false
}
