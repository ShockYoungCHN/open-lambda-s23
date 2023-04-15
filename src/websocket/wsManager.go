package websocket

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

type WsManager struct {
	clients    sync.Map
	pubSubType string
	pubSub     PubSub        // the pubsub that is used to publish and subscribe to topics, e,g redis, kafka, etc
	redis      *redis.Client // the db that stores all the connection id. todo: make this a generic interface
}

func NewWsManager() *WsManager {
	redisClient := redis.NewClient(
		&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})

	return &WsManager{
		clients:    sync.Map{},
		pubSubType: "redis",
		pubSub:     newPubSub(),
		redis:      redisClient,
	}
}

func (manager *WsManager) RegisterClient(id uuid.UUID, wsClient *Client) {
	manager.clients.Store(id.String(), wsClient)

	ctx := context.Background()
	err := manager.redis.SAdd(ctx, "clients", id.String(), 0).Err()
	if err != nil {
		panic(err)
	}
}

func (manager *WsManager) UnregisterClient(id uuid.UUID) {
	manager.clients.Delete(id.String())

	ctx := context.Background()
	err := manager.redis.SRem(ctx, "clients", id.String(), 0).Err()
	if err != nil {
		panic(err)
	}
}

func (manager *WsManager) GetClient(id uuid.UUID) (*Client, bool) {
	client, ok := manager.clients.Load(id.String())

	if ok {
		return client.(*Client), true
	}
	return nil, false
}

func (manager *WsManager) subscribe(channel ...string) *<-chan PubSubMessage {
	ch := manager.pubSub.Subscribe(channel...)

	msgChan := make(chan PubSubMessage, 10)

	switch ch := ch.(type) {
	case <-chan *redis.Message:
		go func() {
			for {
				select {
				case redisMsg := <-ch:
					msg := PubSubMessage{
						Topic:   redisMsg.Channel,
						Message: redisMsg.Payload,
					}
					msgChan <- msg
				}
			}
		}()
		// todo: add sarama kafka support
		/*	case *sarama.ConsumerMessage:
			go func() {
				for {
					select {
					case kafkaMsg := <-ch:
						msg := PubSubMessage{
							Topic:   kafkaMsg.Topic,
							Message: kafkaMsg.Value,
						}
						msgChan <- msg
					}
				}
			}()*/
	default:
		panic(fmt.Sprintf("Invalid channel type: %T", ch))
	}
	readonlyChan := (<-chan PubSubMessage)(msgChan)
	return &readonlyChan
}

func (manager *WsManager) unsubscribe(pubSub *redis.PubSub, channel ...string) error {
	for _, c := range channel {
		switch {
		case manager.pubSubType == "redis":
			if err := manager.pubSub.Unsubscribe(pubSub, c); err != nil {
				return err
			}
		case manager.pubSubType == "kafka":
			// TODO: implement Kafka unsubscribe
			return errors.New("Kafka unsubscribe not implemented")
		default:
			return fmt.Errorf("Invalid channel name: %s", c)
		}
	}
	return nil
}

func (manager *WsManager) publish(v interface{}, channel ...string) {
	log.Println("publishing to channel: ", channel)
	manager.pubSub.Publish(v, channel...)
}

func (manager *WsManager) getAllWsConn() []string {
	ctx := context.Background()
	members, err := manager.redis.SMembers(ctx, "clients").Result()
	if err != nil {
		panic(err)
	}
	return members
}

// TODO: monitor redis pubsub channels and send messages to clients
/*
// send sends the http request to the lambda server and sends the response to the client
func send(client *Client) {

	err = wsutil.WriteServerText(client.conn, body)
	if err != nil {
		log.Printf("failed to write WebSocket message: %s", err)
	}
}*/

func (manager *WsManager) monitorChannels() {
	log.Println("monitoring channels for clients")
	for {
		var size int

		manager.clients.Range(func(key, value interface{}) bool { // 使用.Range()替换.IterCb()
			client := value.(*Client)
			size++
			if client.msgChan != nil {
				log.Println("client has a message channel")
			}

			if client.msgChan == nil {
				return true
			}

			select {
			case msg := <-*(client.msgChan):
				err := wsutil.WriteServerText(client.conn, []byte(msg.Message.(string)))
				log.Println("Sending message to client: ", msg.Message)
				if err != nil {
					log.Printf("failed to write WebSocket message: %s", err)
				}
			default:
			}
			return true
		})
		time.Sleep(2 * time.Second)
		log.Println("size of clients: ", size)
	}
}
