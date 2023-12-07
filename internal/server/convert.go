package server

import (
	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/apis/chat"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapToAPIMessage(message *Message) *chat.Message {
	return &chat.Message{
		Number:    int64(message.Number),
		RoomId:    message.RoomID,
		UserId:    message.UserID,
		Text:      message.Text,
		CreatedAd: timestamppb.New(message.CreatedAt),
	}
}

func mapToAPIMessageList(messages []*Message) *chat.MessageList {
	apiMessages := make([]*chat.Message, len(messages))
	for i, message := range messages {
		apiMessages[i] = mapToAPIMessage(message)
	}
	return &chat.MessageList{
		Messages: apiMessages,
	}
}
