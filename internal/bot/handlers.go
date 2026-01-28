package bot

import (
	"context"
	"errors"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	game    *service.GameService
	repo    *postgres.SituationRepository
	adminID int64

	// Состояние добавления ситуации админом
	adminState *AdminState
}

type AdminState struct {
	IsAdding      bool
	SituationID   int
	WaitingAnswer bool
}

func NewHandler(bot *tgbotapi.BotAPI, game *service.GameService, repo *postgres.SituationRepository, adminID int64) *Handler {
	return &Handler{
		bot:        bot,
		game:       game,
		repo:       repo,
		adminID:    adminID,
		adminState: &AdminState{},
	}
}

func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(ctx, update.Message)
	} else if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
	}
}

func (h *Handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			h.cmdStart(ctx, msg)
		case "add":
			h.cmdAdd(ctx, msg)
		case "reset":
			h.cmdReset(ctx, msg)
		case "stats":
			h.cmdStats(ctx, msg)
		case "help":
			h.cmdHelp(ctx, msg)
		default:
			h.sendText(msg.Chat.ID, "Неизвестная команда. Используйте /help")
		}
		return
	}

	if msg.Photo != nil && h.isAdmin(msg.From.ID) {
		h.handleAdminPhoto(ctx, msg)
		return
	}

	if msg.Text != "" && h.isAdmin(msg.From.ID) && h.adminState.WaitingAnswer {
		h.handleAdminAnswer(ctx, msg)
		return
	}
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	
	callback := tgbotapi.NewCallback(cb.ID, "")
	h.bot.Request(callback)

	switch cb.Data {
	case "more_photo":
		h.cbMorePhoto(ctx, cb)
	case "show_answer":
		h.cbShowAnswer(ctx, cb)
	case "next_round":
		h.cbNextRound(ctx, cb)
	case "finish_add":
		h.cbFinishAdd(ctx, cb)
	case "confirm_reset":
		h.cbConfirmReset(ctx, cb)
	case "cancel_reset":
		h.cbCancelReset(ctx, cb)
	}
}

// === Команды ===

func (h *Handler) cmdStart(ctx context.Context, msg *tgbotapi.Message) {
	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			h.sendText(msg.Chat.ID, "😔 Нет доступных ситуаций.\n\nАдминистратор может добавить их командой /add")
			return
		}
		log.Printf("Error starting round: %v", err)
		h.sendText(msg.Chat.ID, "Произошла ошибка при запуске игры")
		return
	}

	h.sendGamePhoto(ctx, msg.Chat.ID, photo.FileID)
}

func (h *Handler) cmdAdd(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(msg.From.ID) {
		h.sendText(msg.Chat.ID, "⛔ Эта команда доступна только администратору")
		return
	}

	h.adminState.IsAdding = true
	h.adminState.WaitingAnswer = true
	h.adminState.SituationID = 0

	h.sendText(msg.Chat.ID, "📝 Введите правильный ответ для новой ситуации:")
}

func (h *Handler) cmdReset(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(msg.From.ID) {
		h.sendText(msg.Chat.ID, "⛔ Эта команда доступна только администратору")
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Вы уверены, что хотите сбросить игру?\n\nВсе ситуации снова станут доступны для игры.")
	reply.ReplyMarkup = ConfirmResetKeyboard()
	h.bot.Send(reply)
}

func (h *Handler) cmdStats(ctx context.Context, msg *tgbotapi.Message) {
	total, used, remaining, err := h.game.GetStats(ctx)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		h.sendText(msg.Chat.ID, "Ошибка получения статистики")
		return
	}

	text := fmt.Sprintf("📊 *Статистика игры*\n\n"+
		"Всего ситуаций: %d\n"+
		"Сыграно: %d\n"+
		"Осталось: %d", total, used, remaining)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

func (h *Handler) cmdHelp(ctx context.Context, msg *tgbotapi.Message) {
	text := `🎮 *Photo Quiz Bot*

*Команды игры:*
/start — начать игру (показать ситуацию)
/stats — статистика игры

*Команды администратора:*
/add — добавить новую ситуацию
/reset — сбросить игру (все ситуации снова доступны)

*Как играть:*
1. Нажмите /start
2. Смотрите на фото и угадывайте ситуацию
3. Кнопка "Ещё" покажет еще фотографии по ситуации
4. "Правильный ответ" покажет ответ
5. "Следующий ход" — переход к новой ситуации`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

// === Обработка фото и ответов от админа ===

func (h *Handler) handleAdminAnswer(ctx context.Context, msg *tgbotapi.Message) {
	
	situationID, err := h.repo.CreateSituation(ctx, msg.Text)
	if err != nil {
		log.Printf("Error creating situation: %v", err)
		h.sendText(msg.Chat.ID, "Ошибка создания ситуации")
		h.adminState.IsAdding = false
		h.adminState.WaitingAnswer = false
		return
	}

	h.adminState.SituationID = situationID
	h.adminState.WaitingAnswer = false

	h.sendText(msg.Chat.ID, fmt.Sprintf("✅ Ситуация создана (ID: %d)\n\n📷 Теперь отправьте фотографии (от 1 до 5):", situationID))
}

func (h *Handler) handleAdminPhoto(ctx context.Context, msg *tgbotapi.Message) {
	if !h.adminState.IsAdding || h.adminState.SituationID == 0 {
		h.sendText(msg.Chat.ID, "Сначала используйте /add для создания новой ситуации")
		return
	}

	photo := msg.Photo[len(msg.Photo)-1]

	count, err := h.repo.CountPhotos(ctx, h.adminState.SituationID)
	if err != nil {
		log.Printf("Error counting photos: %v", err)
		h.sendText(msg.Chat.ID, "Ошибка при добавлении фото")
		return
	}

	if count >= 5 {
		h.sendText(msg.Chat.ID, "⚠️ Достигнут лимит в 5 фотографий для этой ситуации")
		return
	}

	// Добавляем фото
	err = h.repo.AddPhoto(ctx, h.adminState.SituationID, photo.FileID)
	if err != nil {
		log.Printf("Error adding photo: %v", err)
		h.sendText(msg.Chat.ID, "Ошибка при добавлении фото")
		return
	}

	count++
	text := fmt.Sprintf("✅ Фото %d добавлено\n\nМожете отправить ещё (максимум 5) или нажмите кнопку для завершения", count)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = AdminKeyboard(h.adminState.SituationID)
	h.bot.Send(reply)
}

func (h *Handler) cbMorePhoto(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	photo, err := h.game.NextPhoto(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoMorePhotos) {
			h.answerCallback(cb.ID, "Больше нет фотографий")
			return
		}
		if errors.Is(err, service.ErrGameNotStarted) {
			h.answerCallback(cb.ID, "Сначала начните игру: /start")
			return
		}
		log.Printf("Error getting next photo: %v", err)
		return
	}

	h.sendGamePhoto(ctx, cb.Message.Chat.ID, photo.FileID)
}

func (h *Handler) cbShowAnswer(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	answer, err := h.game.GetAnswer(ctx)
	if err != nil {
		if errors.Is(err, service.ErrGameNotStarted) {
			h.answerCallback(cb.ID, "Сначала начните игру: /start")
			return
		}
		log.Printf("Error getting answer: %v", err)
		return
	}

	text := fmt.Sprintf("🎯 *Правильный ответ:*\n\n%s", answer)
	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, text)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

func (h *Handler) cbNextRound(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	
	err := h.game.FinishRound(ctx)
	if err != nil && !errors.Is(err, service.ErrGameNotStarted) {
		log.Printf("Error finishing round: %v", err)
	}

	// Начинаем новый
	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			h.sendText(cb.Message.Chat.ID, "🎉 Поздравляем! Все ситуации сыграны!\n\nАдминистратор может сбросить игру командой /reset")
			return
		}
		log.Printf("Error starting new round: %v", err)
		return
	}

	h.sendGamePhoto(ctx, cb.Message.Chat.ID, photo.FileID)
}

func (h *Handler) cbFinishAdd(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	count, _ := h.repo.CountPhotos(ctx, h.adminState.SituationID)

	h.adminState.IsAdding = false
	h.adminState.SituationID = 0
	h.adminState.WaitingAnswer = false

	h.sendText(cb.Message.Chat.ID, fmt.Sprintf("✅ Ситуация сохранена с %d фото!", count))
}

func (h *Handler) cbConfirmReset(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	err := h.game.ResetGame(ctx)
	if err != nil {
		log.Printf("Error resetting game: %v", err)
		h.sendText(cb.Message.Chat.ID, "Ошибка сброса игры")
		return
	}

	h.sendText(cb.Message.Chat.ID, "✅ Игра сброшена! Все ситуации снова доступны.")
}

func (h *Handler) cbCancelReset(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.sendText(cb.Message.Chat.ID, "❌ Сброс отменён")
}

func (h *Handler) sendGamePhoto(ctx context.Context, chatID int64, fileID string) {
	current, total, err := h.game.GetCurrentPhotoInfo()
	hasMore := err == nil && current < total

	caption := ""
	if err == nil {
		caption = fmt.Sprintf("📷 Фото %d из %d", current, total)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(fileID))
	photo.Caption = caption
	photo.ReplyMarkup = GameKeyboard(hasMore)
	h.bot.Send(photo)
}

func (h *Handler) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	h.bot.Send(msg)
}

func (h *Handler) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	callback.ShowAlert = true
	h.bot.Request(callback)
}

func (h *Handler) isAdmin(userID int64) bool {
	return userID == h.adminID
}