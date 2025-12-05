package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
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

type ClientRPC struct {
	id string
}

func (c *ClientRPC) Receive(args MessageArgs, _ *struct{}) error {
	// print incoming message (from other clients or system)
	fmt.Printf("\n%s\n> ", formatIncoming(args))
	return nil
}

func formatIncoming(m MessageArgs) string {
	// if it's a system join/leave message it is already formatted as "User X joined"
	// otherwise it will be "Sender: text"
	return m.Text
}

func dialWithRetry(addr string) (*rpc.Client, error) {
	var client *rpc.Client
	var err error
	backoff := time.Second
	for i := 0; i < 5; i++ {
		client, err = rpc.Dial("tcp", addr)
		if err == nil {
			return client, nil
		}
		log.Printf("dial error: %v; retrying in %v", err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}
	return nil, err
}

func printHistory(h HistoryReply) {
	fmt.Println("--- Chat history ---")
	for _, m := range h.Messages {
		fmt.Println(m)
	}
	fmt.Println("--------------------")
}

func main() {
	serverAddr := flag.String("addr", "127.0.0.1:1234", "server address")
	name := flag.String("name", "anon", "your display name")
	flag.Parse()

	// start a small RPC server for receiving broadcasts
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("client listen: %v", err)
	}
	clientRPC := &ClientRPC{id: *name}
	if err := rpc.RegisterName("Client", clientRPC); err != nil {
		log.Fatalf("register client rpc: %v", err)
	}
	go rpc.Accept(listener) // serve callbacks

	localAddr := listener.Addr().String()

	// connect to central server and register
	server, err := dialWithRetry(*serverAddr)
	if err != nil {
		log.Fatalf("cannot connect to server: %v", err)
	}
	// register (server will dial back to our local RPC)
	if err := server.Call("ChatServer.Register", RegisterArgs{ID: *name, Addr: localAddr}, &struct{}{}); err != nil {
		log.Fatalf("register failed: %v", err)
	}
	fmt.Printf("Connected to %s as %s. Type messages and press Enter. Type 'history' to fetch history, 'exit' to quit.\n", *serverAddr, *name)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}
		text := strings.TrimSpace(line)
		if text == "exit" {
			_ = server.Call("ChatServer.Unregister", RegisterArgs{ID: *name, Addr: localAddr}, &struct{}{})
			fmt.Println("bye")
			break
		}
		if text == "history" {
			var h HistoryReply
			if err := server.Call("ChatServer.History", struct{}{}, &h); err != nil {
				log.Printf("history call error: %v", err)
				continue
			}
			printHistory(h)
			continue
		}

		// send message to server (server will broadcast to others)
		args := MessageArgs{Sender: *name, Text: fmt.Sprintf("%s: %s", *name, text)}
		var reply HistoryReply
		if err := server.Call("ChatServer.Send", args, &reply); err != nil {
			log.Printf("send error: %v", err)
			// try reconnect once
			server.Close()
			server, err = dialWithRetry(*serverAddr)
			if err != nil {
				log.Printf("reconnect failed: %v", err)
				continue
			}
			if err := server.Call("ChatServer.Send", args, &reply); err != nil {
				log.Printf("send after reconnect failed: %v", err)
				continue
			}
		}
		// print updated history locally (includes own message)
		printHistory(reply)
	}

	// cleanup
	server.Close()
	listener.Close()
}
