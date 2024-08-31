package scraper

import (
	"github.com/zelenin/go-tdlib/client"
)

type TgClient interface {
	GetChatHistory(req *client.GetChatHistoryRequest) (*client.Messages, error)
	GetListener() *client.Listener
}
