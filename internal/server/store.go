package server

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/google/uuid"
)

const (
	maxMessages  = 1000
	maxRetention = 7 * 24 * time.Hour
)

var ErrNoRoom = fmt.Errorf("room not found")

type Store struct {
	rdb *redis.Client

	roomHubs   map[string]*RoomHub
	roomHubsMx sync.RWMutex

	messages   map[string][]*Message
	messagesMx sync.Mutex
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{
		rdb:      rdb,
		roomHubs: make(map[string]*RoomHub),
		messages: make(map[string][]*Message),
	}
}

func (s *Store) CreateRoom(userID string, name string) (*Room, error) {
	room := &Room{
		ID:      uuid.New().String(),
		OwnerID: userID,
		Name:    name,
	}
	err := s.addRoom(room)
	if err != nil {
		return nil, fmt.Errorf("failed to save room info: %w", err)
	}

	return room, nil
}

func (s *Store) GetRoomHub(ctx context.Context, id string) (*RoomHub, error) {
	room, err := s.getRoom(id)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrNoRoom
	}

	return s.getOrCreateRoomHub(ctx, room)
}

func (s *Store) getRoom(id string) (*Room, error) {
	ctx := context.Background()

	// Получаем JSON из Redis по ключу
	key := fmt.Sprintf("rooms:%s:info", id)
	var room Room
	err := s.rdb.Get(ctx, key).Scan(&room)
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNoRoom
		}
		return nil, fmt.Errorf("failed to load room info from Redis: %w", err)
	}

	if err != nil {
		if err == redis.Nil {
			return nil, ErrNoRoom
		}
		return nil, fmt.Errorf("failed to unmarshal room info: %w", err)
	}
	return &room, nil
}

func (s *Store) addRoom(room *Room) error {
	ctx := context.Background()

	key := fmt.Sprintf("rooms:%s:info", room.ID)
	err := s.rdb.Set(ctx, key, room, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save room info to Redis: %w", err)
	}

	return nil
}

func (s *Store) getOrCreateRoomHub(ctx context.Context, room *Room) (*RoomHub, error) {
	s.roomHubsMx.RLock()
	hub := s.roomHubs[room.ID]
	s.roomHubsMx.RUnlock()

	if hub != nil {
		return hub, nil
	}

	s.roomHubsMx.Lock()
	defer s.roomHubsMx.Unlock()

	// Double-checked locking
	hub = s.roomHubs[room.ID]
	if hub != nil {
		return hub, nil
	}

	hub, err := NewRoomHub(ctx, room, s)
	if err != nil {
		return nil, err
	}

	s.roomHubs[room.ID] = hub

	return hub, nil
}

func (s *Store) loadMessages(ctx context.Context, roomID string) (messages []*Message, lastNumber int, err error) {
	tx := s.rdb.TxPipeline()

	getMessagesCmd := tx.ZRevRangeByScore(ctx, s.roomMessagesKey(roomID), &redis.ZRangeBy{
		Count: int64(maxMessages),
		Min:   fmt.Sprintf("%d", time.Now().Add(-maxRetention).UnixNano()),
		Max:   "+inf",
	})

	getMessageNumberCmd := tx.Get(ctx, s.messageNumberKey(roomID))

	if _, err := tx.Exec(ctx); err != nil && err != redis.Nil {
		return nil, 0, fmt.Errorf("exec pipeline: %w", err)
	}
	err = getMessagesCmd.ScanSlice(&messages)
	if err != nil {
		return nil, 0, fmt.Errorf("get messages: %w", err)
	}

	lastNumber, err = getMessageNumberCmd.Int()
	switch {
	case err == redis.Nil:
		lastNumber = -1
	case err != nil:
		return nil, 0, fmt.Errorf("get message number: %w", err)

	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Number < messages[j].Number
	})
	return messages, lastNumber, nil
}

func (s *Store) SaveMessage(ctx context.Context, message *Message) error {
	tx := s.rdb.TxPipeline()

	addMessageCmd := tx.ZAdd(ctx, s.roomMessagesKey(message.RoomID), redis.Z{
		Score:  float64(message.CreatedAt.UnixNano()),
		Member: message,
	})

	updateNumberCmd := tx.Set(ctx, s.messageNumberKey(message.RoomID), message.Number, 0)
	if _, err := tx.Exec(ctx); err != nil {
		return fmt.Errorf("exec pipeline: %w", err)
	}

	if err := addMessageCmd.Err(); err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	if err := updateNumberCmd.Err(); err != nil {
		return fmt.Errorf("update message number: %w", err)
	}
	return nil
}

func (s *Store) roomMessagesKey(roomID string) string {
	return fmt.Sprintf("room:{%s}:info", roomID)
}

func (s *Store) messageNumberKey(roomID string) string {
	return fmt.Sprintf("room:{%s}:message", roomID)
}

type Connection struct {
	UserID     string
	RoomID     string
	Unread     []*Message
	MessagesCh chan *Message
	Disconnect func()
}
