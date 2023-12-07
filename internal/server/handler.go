package server

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/apis/chat"
	"golang.org/x/exp/slog"
)

var _ chat.ChatServiceServer = (*ChatServer)(nil)

type ChatServer struct {
	chat.UnimplementedChatServiceServer
	store *Store
}

func NewChatServer(store *Store) *ChatServer {
	return &ChatServer{
		store: store,
	}
}

func (s *ChatServer) CreateRoom(_ context.Context, request *chat.CreateRoomRequest) (*chat.CreateRoomResponse, error) {
	room, err := s.store.CreateRoom(request.UserId, request.Name)
	if err != nil {
		return nil, fmt.Errorf("create room: %s", err)
	}
	return &chat.CreateRoomResponse{
		RoomId: room.ID,
	}, nil
}

func (s *ChatServer) Connect(stream chat.ChatService_ConnectServer) error {
	in, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("receive first message: %w", err)
	}

	connectRoom, ok := in.Payload.(*chat.ConnectRequest_ConnectRoom_)
	if !ok {
		return fmt.Errorf("unexpected payload type for first message: %T", in.Payload)
	}

	hub, err := s.store.GetRoomHub(stream.Context(), connectRoom.ConnectRoom.RoomId)
	if err != nil {
		return fmt.Errorf("get room hub: %w", err)
	}

	connection := hub.Connect(connectRoom.ConnectRoom.UserId, int(connectRoom.ConnectRoom.LastReadMessageNumber))
	defer connection.Disconnect()

	if err := stream.Send(&chat.ConnectResponse{
		Payload: &chat.ConnectResponse_MessageList{
			MessageList: mapToAPIMessageList(connection.Unread),
		},
	}); err != nil {
		return fmt.Errorf("send unread messages: %w", err)
	}

	go func() {
		for message := range connection.MessagesCh {
			if err = stream.Send(&chat.ConnectResponse{
				Payload: &chat.ConnectResponse_Message{
					Message: mapToAPIMessage(message),
				},
			}); err != nil {
				slog.Error("send message", "error", err)
			}
		}
	}()

	for {
		in, err = stream.Recv()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return fmt.Errorf("receive: %w", err)
		case stream.Context().Err() != nil:
			return stream.Context().Err()
		}

		switch payload := in.Payload.(type) {
		case *chat.ConnectRequest_SendMessage_:
			msg := &Message{
				UserID:    connection.UserID,
				RoomID:    connection.RoomID,
				Text:      payload.SendMessage.Text,
				CreatedAt: time.Now(),
			}
			if err = hub.ReceiveMessage(stream.Context(), msg); err != nil {
				return fmt.Errorf("receive message: %w", err)
			}
		default:
			return fmt.Errorf("unexpected payload type: %T", payload)
		}
	}
}
