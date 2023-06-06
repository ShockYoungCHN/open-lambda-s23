package websocket

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/consul/api"
	"github.com/open-lambda/open-lambda/ol/boss"
	"github.com/urfave/cli"
	"log"
	"net"
	"time"
)

var conf = boss.Config{}

var ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")

var manager *WsManager

type WsPacket struct {
	Action string `json:"action"` // route to the corresponding handler, for now it's only `run`
	Target string `json:"target"`
	Body   string `json:"body"`
}

type deadliner struct {
	net.Conn
	t time.Duration
}

type HandlerFunc func(interface{})

var routeMap = map[string]HandlerFunc{
	//"sub": func(v interface{}) { sub(v.(*Client)) }, //deprecated pubsub
	"run": func(v interface{}) { run(v.(*Client)) }, // run something, lambda func is specified in the target
}

// router parse the request and call the corresponding handler
func router(client *Client) {
	packet := client.wsPacket
	/* pubsub deprecated
	// if the packet is run, then set the pubTopics to the target, maybe better ways to organize this
		if packet.Action == "run" {
			client.pubTopics = packet.Target
		}*/
	handler, ok := routeMap[packet.Action]
	if !ok {
		log.Println("Unknown action:", packet.Action)
		return
	}
	log.Printf("route to: %s", packet.Action)
	handler(client)
}

//func sub(client *Client) {
//	client.sub()
//}

func run(client *Client) {
	client.run()
}

// wsHandler upgrade the http connection to websocket
func wsHandler(conn net.Conn) {
	safeConn := deadliner{conn, *ioTimeout}
	// Upgrade the connection to WebSocket
	hs, err := ws.Upgrade(safeConn)

	if err != nil {
		log.Printf("%s: upgrade error: %v", nameConn(conn), err)
		conn.Close()
		return
	}
	log.Printf("%s: established websocket connection: %v", nameConn(conn), hs)

	client := &Client{conn: conn}
	client.onConnect()
}

// polling poll the events from the epoll and call the router to handle it
func polling() {
	for {
		clients, err := clientEpoll.Wait()
		if err != nil {
			log.Println("epoll error:", err)
		}
		for _, client := range clients {
			op, r, err := wsutil.NextReader(client.conn, ws.StateServerSide)
			if err != nil {
				fmt.Println("error:", err)
				client.onDisconnect()
				continue
			}

			// todo: handle ping/pong ... more events
			if op.OpCode.IsControl() {
				switch op.OpCode {
				case ws.OpClose:
					client.onDisconnect()
				}
				continue
			}

			req := &(client.wsPacket)
			decoder := json.NewDecoder(r)
			if err := decoder.Decode(req); err != nil {
				log.Println("Error parsing packet:", err)
				continue
			}
			client.event = &Event{
				Context: &Context{Id: client.id},
				Body:    &req.Body,
			}
			go router(client)
		}
	}
}

// todo: integrate loadConf with the boss package
func loadConf() {
	conf.Boss_port = "5000"
	/*	var content []byte
		var err error
		for { // blocking read the boss.json file
			content, err = ioutil.ReadFile("boss.json")
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		err = json.Unmarshal(content, &conf)
		if err != nil {
			log.Fatal(err)
		}*/
}

func reigsterService() {
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Define a new service
	serviceDef := &api.AgentServiceRegistration{
		ID:      "ws-server1",
		Name:    "websocket-server",
		Port:    8080,
		Address: "127.0.0.1",
	}

	// Register the service
	if err := client.Agent().ServiceRegister(serviceDef); err != nil {
		log.Fatal(err)
	}
}

// todo: when to close the connection?
func Start(ctx *cli.Context) error {
	loadConf()
	host := ctx.String("host")
	port := ctx.String("port")
	url := host + ":" + port
	log.Println("ws-api listening on " + url)
	manager = NewWsManager()
	clientEpoll, _ = MkEpoll()
	go polling()
	//go manager.monitorChannels()
	go manager.startInternalApi()

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	reigsterService()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
		}
		go wsHandler(conn)
	}
	return nil
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
