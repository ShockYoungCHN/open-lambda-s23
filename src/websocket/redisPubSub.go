package websocket

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type redisPubSub struct {
	client *redis.Client
	ctx    context.Context
}

func newPubSub() redisPubSub {
	return redisPubSub{
		redis.NewClient(
			&redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			}),
		context.Background(),
	}
}

func (p redisPubSub) Subscribe(topics ...string) interface{} {
	redisCh := p.client.Subscribe(p.ctx, topics...).Channel()
	return redisCh
}

// do it make snese to publish to multiple channels?
func (p redisPubSub) Publish(v interface{}, channels ...string) {
	for _, channel := range channels {
		p.client.Publish(p.ctx, channel, v)
	}
}

func (p redisPubSub) Unsubscribe(pubSub *redis.PubSub, channels ...string) error {
	err := pubSub.Unsubscribe(p.ctx, channels...)
	return err
}

func (p redisPubSub) GetAllActiveChannels() ([]string, error) {
	result, err := p.client.PubSubChannels(p.ctx, "*").Result()
	return result, err
}
