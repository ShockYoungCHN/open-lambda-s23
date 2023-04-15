package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"net/http"
)

type Client struct {
	conn        net.Conn
	id          uuid.UUID
	target      string
	pubTopics   string // channel or topic to publish to, currently only one, but could be multiple in the future
	msgChan     *<-chan PubSubMessage
	requestPath string
	PubSub      *redis.PubSub
	wsPacket    WsPacket
}

func (client *Client) onConnect() {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	client.id = id
	manager.RegisterClient(client.id, client)
}

func (client *Client) onClose() {
	err := client.conn.Close()
	if err != nil {
		return
	}
	manager.UnregisterClient(client.id)
}

func (client *Client) sub() {
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

func (client *Client) run() {
	payload, err := json.Marshal(client.wsPacket)
	body, _, err := sendRequest(payload)
	if err != nil {
		log.Println(err)
		body = []byte(err.Error())
	}
	go client.pub(body)
}

// sendRequest sends an HTTP request to the lambda server and returns the response body
func sendRequest(payload []byte) ([]byte, int, error) {
	url := "http://localhost:" + conf.Boss_port + "/run/echo" // todo: get "/run/echo" from wsPacket

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
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
