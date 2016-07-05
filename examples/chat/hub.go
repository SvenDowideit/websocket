// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"gopkg.in/redis.v4"
)

// hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered connections.
	connections map[*Conn]bool

	// Inbound messages from the connections.
	broadcast chan []byte

	// Register requests from the connections.
	register chan *Conn

	// Unregister requests from connections.
	unregister chan *Conn
}

var hub = Hub{
	broadcast:   make(chan []byte),
	register:    make(chan *Conn),
	unregister:  make(chan *Conn),
	connections: make(map[*Conn]bool),
}

func (h *Hub) run() {
    client := redis.NewClient(&redis.Options{
        Addr:     "redis:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })

    pong, err := client.Ping().Result()
    fmt.Println(pong, err)

	pubsub, err := client.Subscribe("chat")
if err != nil {
    panic(err)
}
defer pubsub.Close()
go func() {
	for {
		message, err := pubsub.ReceiveMessage()
		if err != nil {
		    panic(err)
		}
		fmt.Printf("redis receive: %s\n", message.Payload)
		for conn := range h.connections {
			select {
			case conn.send <- []byte(message.Payload):
			default:
				close(conn.send)
				delete(hub.connections, conn)
			}
		}
	}
}()


	for {
		select {
		case conn := <-h.register:
			name, err := os.Hostname()
			if err != nil {
			    panic(err)
			}
			message := "Connection: " + name
			err = client.Publish("chat", message).Err()
			if err != nil {
			    panic(err)
			}
			h.connections[conn] = true
		case conn := <-h.unregister:
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.send)
			}
		case message := <-h.broadcast:
			err = client.Publish("chat", string(message)).Err()
			if err != nil {
			    panic(err)
			}
		}
	}
}
