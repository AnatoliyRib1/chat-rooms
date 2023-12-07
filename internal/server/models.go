package server

import (
	"encoding"
	"encoding/json"
	"time"
)

var (
	_ encoding.BinaryMarshaler   = &Message{}
	_ encoding.BinaryUnmarshaler = &Message{}
)

type Message struct {
	Number    int
	UserID    string
	RoomID    string
	Text      string
	CreatedAt time.Time
}

func (m *Message) MarshalBinary() (data []byte, err error) {
	return json.Marshal(m)
}

func (m *Message) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

var (
	_ encoding.BinaryMarshaler   = &Room{}
	_ encoding.BinaryUnmarshaler = &Room{}
)

type Room struct {
	ID      string
	OwnerID string
	Name    string
}

func (r *Room) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
