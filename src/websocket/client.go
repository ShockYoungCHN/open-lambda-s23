package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

type Client struct {
	// todo:below 4 are related to pubsub, maybe deleted
	target    string
	pubTopics string // channel or topic to publish to, currently only one, but could be multiple in the future
	msgChan   *<-chan PubSubMessage
	PubSub    *redis.PubSub

	conn     net.Conn
	id       uuid.UUID
	wsPacket WsPacket
	writeMux sync.Mutex
	event    *Event
}

type Event struct {
	Context *Context `json:"context"`
	Body    *string  `json:"body"`
}

type Context struct {
	Id uuid.UUID `json:"id"`
}

func (client *Client) onConnect() {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	client.id = id
	manager.RegisterClient(client)

	wsutil.WriteServerText(client.conn, []byte(id.String()))
}

func (client *Client) onDisconnect() {
	err := client.conn.Close()
	if err != nil {
		return
	}

	manager.UnregisterClient(client)
}

/*func (client *Client) sub() {
	client.msgChan = manager.subscribe(client.wsPacket.Target)
	log.Printf("%s subscribed to %s \n", client.id.String(), client.wsPacket.Target)
}

func (client *Client) pub(body []byte) {
	manager.publish(body, client.pubTopics)
}

func (client *Client) unsubscribe() {
	err := manager.unsubscribe(client.PubSub, client.target)
	if err != nil {
		return
	}
}
*/

func (client *Client) run() {
	respBody, _, err := client.sendRequest() // todo: respBody seems not to be used
	if err != nil {
		log.Println(err)
		log.Println(respBody)
	}
}

// sendRequest sends an HTTP request to the lambda server and returns the response body
func (client *Client) sendRequest() ([]byte, int, error) {
	url := "http://localhost:" + conf.Boss_port + "/run/" + client.wsPacket.Target

	eventBytes, _ := json.Marshal(client.event)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(eventBytes))
	if err != nil {
		return nil, -1, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	var respBuf bytes.Buffer
	_, err = io.Copy(&respBuf, resp.Body)
	if err != nil {
		return nil, -1, err
	}

	return respBuf.Bytes(), resp.StatusCode, nil
}
