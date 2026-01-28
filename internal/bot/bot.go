package bot

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	handler *Handler
}

func New(token string, game *service.GameService, repo *postgres.SituationRepository, adminID int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	handler := NewHandler(api, game, repo, adminID)

	return &Bot{
		api:     api,
		handler: handler,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Bot started, waiting for updates...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Bot stopping...")
			return ctx.Err()
		case update := <-updates:
			go b.handler.HandleUpdate(ctx, update)
		}
	}
}