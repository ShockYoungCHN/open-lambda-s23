package websocket

import (
	"golang.org/x/sys/unix"
	"log"
	"net"
	"reflect"
	"sync"
)

// epoll is a wrapper around the epoll system call.
// It is used to wait for events on a set of file descriptors.
type epoll struct {
	fd      int
	clients map[int]*Client
	lock    *sync.RWMutex
}

func MkEpoll() (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &epoll{
		fd:      fd,
		lock:    &sync.RWMutex{},
		clients: make(map[int]*Client),
	}, nil
}

func (e *epoll) Add(client *Client) error {
	conn := client.conn
	// Extract file descriptor associated with the connection
	fd := websocketFD(conn)
	event := &unix.EpollEvent{
		Events: unix.POLLIN | unix.POLLHUP,
		Fd:     int32(fd),
	}
	err := unix.EpollCtl(e.fd, unix.EPOLL_CTL_ADD, fd, event)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.clients[fd] = client
	if len(e.clients)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.clients))
	}
	return nil
}

func (e *epoll) Remove(client *Client) error {
	conn := client.conn
	fd := websocketFD(conn)
	err := unix.EpollCtl(e.fd, unix.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.clients, fd)
	if len(e.clients)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.clients))
	}
	return nil
}

func (e *epoll) Wait() ([]*Client, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(e.fd, events, 100)
	if err != nil {
		return nil, err
	}

	var clients []*Client
	for i := 0; i < n; i++ {
		client := e.clients[int(events[i].Fd)]
		clients = append(clients, client)
	}
	return clients, nil
}

func websocketFD(conn net.Conn) int {
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}
