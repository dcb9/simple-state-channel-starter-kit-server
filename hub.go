package main

import (
	"encoding/json"
	"fmt"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Inbound messages from the clients.
	inboundMsg chan *InboundMsg

	// Unregister requests from clients.
	unregister chan *Client

	channels map[uint][]*Client
}

func newHub() *Hub {
	return &Hub{
		inboundMsg: make(chan *InboundMsg),
		unregister: make(chan *Client),
		channels:   make(map[uint][]*Client),
	}
}

func (h *Hub) register(client *Client, channelId uint) {
	if h.channels[channelId] == nil {
		h.channels[channelId] = make([]*Client, 0, 2)
	}

	found := false
	clients := h.channels[channelId]
	for _, c := range clients {
		if c == client {
			found = true
			fmt.Println(c, client)
		}
	}

	if !found {
		fmt.Println("add a client to a channel")
		h.channels[channelId] = append(clients, client)
	}
}

func (h *Hub) payment(senderWsClient *Client, payment *PaymentMsg) error {
	paymentBytes , err := json.Marshal(payment)
	if err != nil {
		return err
	}

	for _, client := range h.channels[payment.ChannelID] {
		if client == senderWsClient {
			continue
		}

		select {
		case client.send <- paymentBytes:
		default:
			close(client.send)
			h.unregister <- client
		}
	}

	return nil
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.unregister:
			for channelID, clients := range h.channels {
				for i, c := range clients {
					if c == client {
						// remove client
						fmt.Println("remove a client from a channel")
						h.channels[channelID] = append(clients[:i], clients[i+1:]...)
					}
				}
			}
		case inboundMsg := <-h.inboundMsg:
			switch inboundMsg.MsgType {
			case RegisterMsgType:
				var msg RegisterMsg
				if err := json.Unmarshal(inboundMsg.Data, &msg); err != nil {
					fmt.Println("unmarshal register msg err")
					fmt.Println("data:", string(inboundMsg.Data))
					fmt.Println("err:", err)
					continue
				}
				h.register(inboundMsg.client, msg.ChannelID)
			case PaymentMsgType:
				var msg PaymentMsg
				if err := json.Unmarshal(inboundMsg.Data, &msg); err != nil {
					fmt.Println("unmarshal payment msg err")
					fmt.Println("data:", string(inboundMsg.Data))
					fmt.Println("err:", err)
					continue
				}

				h.payment(inboundMsg.client, &msg)
			}
		}
	}
}

const (
	RegisterMsgType = iota
	PaymentMsgType
)

type InboundMsg struct {
	client *Client
	MsgType int `json:"type"`
	Data json.RawMessage `json:"data"`
}

type RegisterMsg struct {
	ChannelID   uint `json:"channelId"`
}

type PaymentMsg struct {
	Payer     string `json:"payer"`
	Recipient string `json:"recipient"`
	ChannelID uint   `json:"channelId"`
	Value     uint64 `json:"value"`
	Sig       struct {
		S string `json:"s"`
		V string `json:"v"`
		R string `json:"r"`
	} `json:"sig"`
	Timestamp uint64 `json:"timestamp"`
}
