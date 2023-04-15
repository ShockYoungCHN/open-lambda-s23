package websocket

import (
	"encoding/json"
	"flag"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/open-lambda/open-lambda/ol/boss"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"net"
	"time"
)

var conf = boss.Config{}

var ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")

var manager *WsManager

type WsPacket struct {
	Action string `json:"action"`
	Target string `json:"target"`
}

type deadliner struct {
	net.Conn
	t time.Duration
}

type HandlerFunc func(interface{})

var routeMap = map[string]HandlerFunc{
	"sub": func(v interface{}) { sub(v.(*Client)) },
	"run": func(v interface{}) { run(v.(*Client)) }, // no route for pub, it is called directly after run
}

// router parse the request and call the corresponding handler
func router(client *Client) {
	packet := client.wsPacket
	// if the packet is run, then set the pubTopics to the target, maybe better ways to organize this
	if packet.Action == "run" {
		client.pubTopics = packet.Target
	}
	client.wsPacket = packet
	handler, ok := routeMap[packet.Action]
	if !ok {
		log.Println("Unknown action:", packet.Action)
		return
	}

	log.Printf("route to: %s", packet.Action)
	handler(client)
}

func sub(client *Client) {
	client.sub()
}

func run(client *Client) {
	client.run()
}

// wsHandler upgrade the http connection to websocket and start the netpoll
// if receive the data, register the ws, then call the router to handle it
func wsHandler(conn net.Conn) {
	safeConn := deadliner{conn, *ioTimeout}
	// Upgrade the connection to WebSocket
	hs, err := ws.Upgrade(safeConn)

	if err != nil {
		log.Printf("%s: upgrade error: %v", nameConn(conn), err)
		conn.Close()
		return
	}
	log.Printf("%s: established websocket connection: %+v", nameConn(conn), hs)

	client := &Client{conn: conn}
	client.onConnect()

	for {
		op, r, err := wsutil.NextReader(client.conn, ws.StateServerSide)
		if err != nil {
			conn.Close()
			return
		}

		// todo: handle ping/pong ... more events
		if op.OpCode.IsControl() {
			switch op.OpCode {
			case ws.OpClose:
				client.onClose()
				return
			}
		}

		req := &(client.wsPacket)
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(req); err != nil {
			log.Println("Error parsing packet:", err)
			return
		}
		go router(client)
	}
}

// todo: integrate loadConf with the boss package
func loadConf() {
	var content []byte
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
	go manager.monitorChannels()

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

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
