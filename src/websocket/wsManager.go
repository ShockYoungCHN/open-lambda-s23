package websocket

import "C"
import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	pb "github.com/open-lambda/open-lambda/ol/websocket/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
	"time"
)

// not sure if epoll should be here or somewhere else
var clientEpoll *epoll

type WsManager struct {
	clients    sync.Map
	pubSubType string
	pubSub     PubSub        // the pubsub that is used to publish and subscribe to topics, e,g redis, kafka, etc
	redis      *redis.Client // the db that stores all the connection id. todo: make this a generic interface
	pb.UnimplementedWsManagerServer
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

// RegisterClient registers a client to the manager and the epoll, call onConnect func in lamda
func (manager *WsManager) RegisterClient(client *Client) {
	manager.clients.Store(client.id.String(), client)
	// register client to epoll
	clientEpoll.Add(client)

	// call onConnect func in lambda
	// set up the event for onConnect
	client.event = &Event{&Context{Id: client.id}, nil}
	client.wsPacket.Target = "onConnect"
	client.sendRequest()
}

func (manager *WsManager) UnregisterClient(client *Client) {
	id := client.id
	manager.clients.Delete(id.String())
	// unregister client from epoll
	clientEpoll.Remove(client)
	// call onDisconnect func in lambda
	client.event = &Event{&Context{Id: client.id}, nil}
	client.wsPacket.Target = "onDisconnect"
	client.sendRequest()
}

func (manager *WsManager) GetClient(id uuid.UUID) (*Client, bool) {
	client, ok := manager.clients.Load(id.String())

	if ok {
		return client.(*Client), true
	}
	return nil, false
}

// deprecated pubsub
/*func (manager *WsManager) subscribe(channel ...string) *<-chan PubSubMessage {
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
*/

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

// todo: a indiv method to send msg back to client connections
/*func (manager *WsManager) postToConnection(msg string, connectionId string) {
	log.Println("posting to connection: ", connectionId)
	client, ok := manager.clients.Load(connectionId)
	if !ok {
		log.Println("Connection not found: ", connectionId)
		return
	}

	wsClient := client.(*Client)
	wsClient.writeMux.Lock()
	defer wsClient.writeMux.Unlock()
	err := wsutil.WriteServerText(wsClient.conn, []byte(msg))
	if err != nil {
		log.Printf("failed to write WebSocket message: %s", err)
	}
}*/

// startInternalApi starts internal APIs for lambda functions
func (manager *WsManager) startInternalApi() {
	log.Println("starting internal APIs")
	ip := "172.29.96.75" //todo: get ip from env
	lis, err := net.Listen("tcp", ip+":50051")
	if err != nil {
		log.Fatalf("internal APIs failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWsManagerServer(s, manager)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("internal APIs failed to serve: %v", err)
	}
}

// PostToConnection is an internal gRPC method for lambda functions to call
// post message to a connection
func (manager *WsManager) PostToConnection(ctx context.Context, req *pb.PostToConnectionRequest) (*pb.PostToConnectionResponse, error) {
	log.Println("posting to connection: ", req.ConnectionId)
	client, ok := manager.clients.Load(req.ConnectionId)
	if !ok {
		return &pb.PostToConnectionResponse{
			Success: false,
			Error:   "Client not found",
		}, nil
	}
	wsClient := client.(*Client)
	err := wsutil.WriteServerText(wsClient.conn, []byte(req.Msg))
	if err != nil {
		return &pb.PostToConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to write WebSocket message: %s", err),
		}, nil
	}
	return &pb.PostToConnectionResponse{
		Success: true,
		Error:   "",
	}, nil
}
