package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/plastinin/photo-quiz-bot/internal/bot"
	"github.com/plastinin/photo-quiz-bot/internal/config"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
)

func main() {

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// БД
	db, err := postgres.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")
	repo := postgres.NewSituationRepository(db)

	// Сервис
	gameService := service.NewGameService(repo)

	//  Бот
	telegramBot, err := bot.New(cfg.BotToken, gameService, repo, cfg.AdminID)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Запускаем бота
	if err := telegramBot.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Bot stopped")
}