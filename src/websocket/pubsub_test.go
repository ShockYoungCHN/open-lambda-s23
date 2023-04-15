package websocket

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestRedisPubSub(t *testing.T) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	pubsub := redisPubSub{
		client: client,
		ctx:    ctx,
	}

	ch1 := "test_channel_1"
	sub := pubsub.Subscribe(ch1)

	msg := "hello world"
	err := client.Publish(ctx, ch1, msg).Err()
	if err != nil {
		t.Fatalf("Failed to publish message to channel %s: %v", ch1, err)
	}

	received := make(chan bool)
	go func() {
		for msgSub := range sub.(<-chan *redis.Message) {
			if msgSub.Payload == msg {
				received <- true
			}
		}
	}()
	select {
	case <-received:
		fmt.Println("Received message successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Failed to receive message within 5 seconds")
	}

	//err = sub.(*redis.PubSub).Unsubscribe(ctx, ch1)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from channel %s: %v", ch1, err)
	}
}
