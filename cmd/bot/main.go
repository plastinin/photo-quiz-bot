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
	"github.com/plastinin/photo-quiz-bot/internal/web"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	repo := postgres.NewSituationRepository(db)
	gameService := service.NewGameService(repo)

	telegramBot, err := bot.New(cfg.BotToken, gameService, repo, cfg.AdminID)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	webServer, err := web.NewServer(":"+cfg.WebPort, repo, cfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
		webServer.Shutdown(context.Background())
	}()

	go func() {
		if err := webServer.Run(); err != nil && err.Error() != "http: Server closed" {
			log.Printf("Web server error: %v", err)
		}
	}()

	if err := telegramBot.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Application stopped")
}