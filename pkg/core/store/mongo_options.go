package store

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoClient struct {
	addr string
	*mongo.Client
}

const uri = "mongodb://%s:%s@%s:%d/?maxPoolSize=20&w=majority"

func NewMongoClient(ctx context.Context, username, password, addr string, port int) (*MongoClient, error) {
	r := &MongoClient{addr: fmt.Sprintf(uri, username, password, addr, port)}

	err := r.start(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mongo client")
		return nil, err
	}

	return r, nil
}

func (r *MongoClient) start(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("start mongo client")

	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(r.addr))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mongo client")
		return err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mongo client")
		return err
	}

	r.Client = client

	return nil
}

func (r *MongoClient) Close(ctx context.Context) error {
	if r.Client == nil {
		return nil
	}
	log.Ctx(ctx).Info().Str("address", r.addr).Msg("closing mongo client")
	return r.Client.Disconnect(ctx)
}

func (r *MongoClient) InsertOne(ctx context.Context, dataBase, col string, doc interface{}) (string, error) {
	res, err := r.Client.Database(dataBase).Collection(col).InsertOne(ctx, doc)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), err
}

func (r *MongoClient) InsertMany(ctx context.Context, dataBase, col string, docs []interface{}) ([]string, error) {
	res, err := r.Client.Database(dataBase).Collection(col).InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(res.InsertedIDs))
	for i, o := range res.InsertedIDs {
		ids[i] = o.(string)
	}
	return ids, err
}
