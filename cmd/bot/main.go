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
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаём контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключаемся к базе данных
	db, err := postgres.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	// Создаём репозиторий и сервис
	repo := postgres.NewSituationRepository(db)
	gameService := service.NewGameService(repo)

	// Создаём веб-сервер
	webServer, err := web.NewServer(":"+cfg.WebPort, repo, cfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	// Создаём и запускаем Telegram бота (передаём webServer для связи)
	telegramBot, err := bot.New(cfg.BotToken, gameService, repo, cfg.AdminID, webServer)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
		webServer.Shutdown(context.Background())
	}()

	// Запускаем веб-сервер в отдельной горутине
	go func() {
		if err := webServer.Run(); err != nil && err.Error() != "http: Server closed" {
			log.Printf("Web server error: %v", err)
		}
	}()

	// Запускаем бота (блокирующий вызов)
	if err := telegramBot.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Application stopped")
}