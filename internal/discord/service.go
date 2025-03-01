package discord

import (
	"github.com/gtuk/discordwebhook"
	"log"
)

type Service struct {
	webhookUrl string
}

func New(webhookUrl string) *Service {
	return &Service{
		webhookUrl: webhookUrl,
	}
}

func (s *Service) Post(content string) {
	if s.webhookUrl == "" {
		return
	}
	var username = "CreateMod.com"
	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(s.webhookUrl, message)
	if err != nil {
		log.Fatal(err)
	}
}
