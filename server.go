package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

type MessageArgs struct {
	Sender string
	Text   string
}

type HistoryReply struct {
	Messages []string
}

type RegisterArgs struct {
	ID   string
	Addr string
}

// ChatServer holds history, connected clients and a broadcast channel.
type ChatServer struct {
	mu        sync.Mutex
	msgs      []string
	clients   map[string]*rpc.Client
	broadcast chan MessageArgs
}

func NewChatServer() *ChatServer {
	c := &ChatServer{
		clients:   make(map[string]*rpc.Client),
		broadcast: make(chan MessageArgs, 100),
	}
	// broadcaster goroutine
	go func() {
		for msg := range c.broadcast {
			// snapshot clients to avoid holding lock during RPC calls
			c.mu.Lock()
			clients := make(map[string]*rpc.Client, len(c.clients))
			for id, cli := range c.clients {
				clients[id] = cli
			}
			c.mu.Unlock()

			for id, cli := range clients {
				if id == msg.Sender {
					continue // no self-echo
				}
				// call each client concurrently
				go func(id string, cli *rpc.Client, m MessageArgs) {
					var reply struct{}
					err := cli.Call("Client.Receive", m, &reply)
					if err != nil {
						// on error remove client
						log.Printf("failed to deliver to %s: %v (removing)", id, err)
						c.mu.Lock()
						cli.Close()
						delete(c.clients, id)
						c.mu.Unlock()
					}
				}(id, cli, msg)
			}
		}
	}()
	return c
}

// Register: client tells server its ID and listening address. Server dials back and stores client RPC.
func (c *ChatServer) Register(args RegisterArgs, reply *struct{}) error {
	cli, err := rpc.Dial("tcp", args.Addr)
	if err != nil {
		return fmt.Errorf("dial client %s at %s: %w", args.ID, args.Addr, err)
	}
	c.mu.Lock()
	c.clients[args.ID] = cli
	joinMsg := fmt.Sprintf("User %s joined", args.ID)
	c.msgs = append(c.msgs, joinMsg)
	c.mu.Unlock()

	// broadcast join to others (no self-echo)
	c.broadcast <- MessageArgs{Sender: args.ID, Text: joinMsg}
	return nil
}

// Unregister: remove client
func (c *ChatServer) Unregister(args RegisterArgs, reply *struct{}) error {
	c.mu.Lock()
	if cli, ok := c.clients[args.ID]; ok {
		cli.Close()
		delete(c.clients, args.ID)
	}
	leaveMsg := fmt.Sprintf("User %s left", args.ID)
	c.msgs = append(c.msgs, leaveMsg)
	c.mu.Unlock()

	c.broadcast <- MessageArgs{Sender: args.ID, Text: leaveMsg}
	return nil
}

// Send: append to history and broadcast to others (no self-echo). Returns full history to caller.
func (c *ChatServer) Send(args MessageArgs, reply *HistoryReply) error {
	entry := fmt.Sprintf("%s: %s", args.Sender, args.Text)
	c.mu.Lock()
	c.msgs = append(c.msgs, entry)
	reply.Messages = append([]string(nil), c.msgs...)
	c.mu.Unlock()

	// broadcast to others
	c.broadcast <- args
	return nil
}

// History: return full history
func (c *ChatServer) History(_ struct{}, reply *HistoryReply) error {
	c.mu.Lock()
	reply.Messages = append([]string(nil), c.msgs...)
	c.mu.Unlock()
	return nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1:1234", "server listen address")
	flag.Parse()

	server := NewChatServer()
	if err := rpc.RegisterName("ChatServer", server); err != nil {
		log.Fatalf("rpc register: %v", err)
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	defer ln.Close()

	log.Printf("Chat server listening on %s", *addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
