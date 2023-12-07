package server

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type RoomHub struct {
	room         *Room
	store        *Store
	lastNumber   int
	mx           sync.RWMutex
	messagesMx   sync.RWMutex
	messages     []*Message
	userChannels map[string]chan *Message
}

func NewRoomHub(ctx context.Context, room *Room, store *Store) (*RoomHub, error) {
	messages, lastNumber, err := store.loadMessages(ctx, room.ID)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	return &RoomHub{
		room:         room,
		store:        store,
		messages:     messages,
		userChannels: make(map[string]chan *Message),
		lastNumber:   lastNumber,
	}, nil
}

func (h *RoomHub) Connect(userID string, lastMessageNumber int) *Connection {
	h.mx.Lock()
	defer h.mx.Unlock()

	connection := &Connection{
		UserID:     userID,
		RoomID:     h.room.ID,
		Unread:     h.getUnreadMessages(lastMessageNumber),
		MessagesCh: make(chan *Message, 4),
		Disconnect: func() {
			h.disconnect(userID)
		},
	}
	h.userChannels[userID] = connection.MessagesCh

	return connection
}

func (h *RoomHub) ReceiveMessage(ctx context.Context, message *Message) error {
	if err := h.saveMessage(ctx, message); err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	h.mx.RLock()
	defer h.mx.RUnlock()

	for _, ch := range h.userChannels {
		ch <- message
	}
	return nil
}

func (h *RoomHub) saveMessage(ctx context.Context, message *Message) error {
	h.messagesMx.Lock()
	defer h.messagesMx.Unlock()

	message.Number = h.lastNumber + 1
	if err := h.store.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	h.messages = append(h.messages, message)
	h.lastNumber++

	return nil
}

func (h *RoomHub) disconnect(userID string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	ch := h.userChannels[userID]
	if ch != nil {
		close(ch)
		delete(h.userChannels, userID)

	}
}

func (h *RoomHub) getUnreadMessages(lastNumber int) []*Message {
	h.messagesMx.RLock()
	defer h.messagesMx.RUnlock()

	switch {
	case lastNumber < 0:
		return h.messages
	case lastNumber >= h.lastNumber:
		return nil
	default:
		idx := sort.Search(len(h.messages), func(i int) bool {
			return h.messages[i].Number > lastNumber
		})
		return h.messages[idx:]
	}
}
