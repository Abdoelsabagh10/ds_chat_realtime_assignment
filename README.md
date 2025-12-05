# Real-Time RPC Chat System — Go (Assignment 05)

This project is a modified version of the RPC-based chat system. The goal is to add real-time broadcasting using Go concurrency, while supporting multiple clients, message history, and join notifications.

## Features

### Real-Time Broadcasting
- When a client sends a message, it is broadcast to all other clients immediately.
- The sender does not receive their own message (no self-echo).

### Client Join Notifications
- When a new client joins, the server notifies all existing clients:
  User [ID] joined

### Message History
- Each client can request the full chat history, including messages and join events.

### Concurrency and Synchronization
- Uses goroutines for concurrent client handling.
- Uses channels for broadcasting messages.
- Uses sync.Mutex to protect shared data such as the client list and history.

## Project Structure

server.go   — Main RPC server that manages clients, broadcasting, and message history  
client.go   — Client application responsible for sending messages and receiving broadcasts  

## Running the System

1. Start the server:
   ```
   go run server.go
   ```

2. Start each client in a separate terminal:
   ```
   go run client.go --name <YourName>
   ```

   Example:
   ```
   go run client.go --name Alice
   go run client.go --name Bob
   ```

## Client Commands

| Command      | Description                                |
|--------------|--------------------------------------------|
| any message  | Sends a message to all other clients        |
| history      | Prints the full chat history                |
| exit         | Disconnects the client                      |

## How It Works

- Each client registers itself with the server when it starts.
- The server maintains a synchronized list of connected clients.
- When a client joins, the server broadcasts a join notification to all other clients.
- When a client sends a message, the server broadcasts it to all clients except the sender.
- Chat history is stored on the server and can be retrieved on demand.

## Assignment Notes

- This repository is newly created specifically for Assignment 05, not the one originally submitted.
- The implementation uses Go RPC, goroutines, channels, and mutexes as required.
- Submit the link to this GitHub repository as your assignment submission.

## Author

Prepared for Assignment 05 — Real-Time RPC Chat System using Go and Concurrent Programming.
