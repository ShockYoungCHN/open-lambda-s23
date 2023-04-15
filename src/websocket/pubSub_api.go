package websocket

import "github.com/go-redis/redis/v8"

type PubSub interface {
	Unsubscribe(pubSub *redis.PubSub, topic ...string) error
	Publish(v interface{}, topics ...string)
	Subscribe(topic ...string) interface{}
}

type PubSubMessage struct {
	Topic   string
	Message interface{}
}
