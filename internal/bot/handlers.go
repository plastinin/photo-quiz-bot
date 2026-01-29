package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
	"github.com/plastinin/photo-quiz-bot/internal/web"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	game    *service.GameService
	repo    *postgres.SituationRepository
	adminID int64
	web     *web.Server

	// Состояние добавления ситуации
	addState   map[int64]*AddSituationState
	addStateMu sync.RWMutex

	// Состояние ввода очков
	scoreState   map[int64]*ScoreInputState
	scoreStateMu sync.RWMutex
}

type AddSituationState struct {
	Answer  string
	Photos  []string
	Waiting bool
}

type ScoreInputState struct {
	PlayerName string
	Waiting    bool
}

func NewHandler(bot *tgbotapi.BotAPI, game *service.GameService, repo *postgres.SituationRepository, adminID int64, webServer *web.Server) *Handler {
	h := &Handler{
		bot:        bot,
		game:       game,
		repo:       repo,
		adminID:    adminID,
		web:        webServer,
		addState:   make(map[int64]*AddSituationState),
		scoreState: make(map[int64]*ScoreInputState),
	}

	// Слушаем события завершения хода из веба
	if webServer != nil {
		go h.listenTurnEndEvents()
	}

	return h
}

func (h *Handler) listenTurnEndEvents() {
	for event := range h.web.Session.TurnEndChan {
		// Отправляем админу запрос на ввод очков
		h.scoreStateMu.Lock()
		h.scoreState[h.adminID] = &ScoreInputState{
			PlayerName: event.PlayerName,
			Waiting:    true,
		}
		h.scoreStateMu.Unlock()

		msg := tgbotapi.NewMessage(h.adminID, fmt.Sprintf("🤑 *Ход завершён!*\n\nИгрок: *%s*\n\nВыберите количество BazuCoin:", event.PlayerName))
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = ScoreKeyboard()
		h.bot.Send(msg)
	}
}

func (h *Handler) Handle(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message != nil {
		h.handleMessage(ctx, update.Message)
	}
}

func (h *Handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Проверяем, ожидаем ли ввод очков
	h.scoreStateMu.RLock()
	scoreState, hasScoreState := h.scoreState[msg.From.ID]
	h.scoreStateMu.RUnlock()

	if hasScoreState && scoreState.Waiting && msg.Text != "" {
		h.handleScoreInput(ctx, msg, scoreState)
		return
	}

	// Проверяем, есть ли активное состояние добавления
	h.addStateMu.RLock()
	state, hasState := h.addState[msg.From.ID]
	h.addStateMu.RUnlock()

	if hasState && state.Waiting {
		h.handleAddState(ctx, msg, state)
		return
	}

	// Обработка команд
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			h.cmdStart(ctx, msg)
		case "add":
			h.cmdAdd(ctx, msg)
		case "reset":
			h.cmdReset(ctx, msg)
		case "delete":
			h.cmdDelete(ctx, msg)
		case "stats":
			h.cmdStats(ctx, msg)
		case "help":
			h.cmdHelp(ctx, msg)
		default:
			h.sendText(msg.Chat.ID, "Неизвестная команда. Используйте /help")
		}
	}
}

func (h *Handler) handleScoreInput(ctx context.Context, msg *tgbotapi.Message, state *ScoreInputState) {
	if !h.isAdmin(msg.From.ID) {
		return
	}

	// Поддержка дробных чисел
	score, err := strconv.ParseFloat(strings.TrimSpace(msg.Text), 64)
	if err != nil {
		h.sendText(msg.Chat.ID, "❌ Введите число (например: 0, 0.5, 1, 1.5, 2, 2.5, 3)")
		return
	}

	// Проверка допустимых значений
	validScores := map[float64]bool{0: true, 0.5: true, 1: true, 1.5: true, 2: true, 2.5: true, 3: true}
	if !validScores[score] {
		h.sendText(msg.Chat.ID, "❌ Допустимые значения: 0, 0.5, 1, 1.5, 2, 2.5, 3")
		return
	}

	// Добавляем очки
	playerName, totalScore, ok := h.web.AddScoreToCurrentPlayer(score)
	if !ok {
		h.sendText(msg.Chat.ID, "❌ Ошибка: нет активной сессии")
		h.clearScoreState(msg.From.ID)
		return
	}

	h.clearScoreState(msg.From.ID)

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ *%s* получает *%.1f* 🤑 BazuCoin!\n\nВсего: *%.1f* 🤑", playerName, score, totalScore))
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

func (h *Handler) clearScoreState(userID int64) {
	h.scoreStateMu.Lock()
	delete(h.scoreState, userID)
	h.scoreStateMu.Unlock()
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	// Отвечаем на callback, чтобы убрать "часики"
	callback := tgbotapi.NewCallback(cb.ID, "")
	h.bot.Request(callback)

	switch {
	case cb.Data == "more_photo":
		h.cbMorePhoto(ctx, cb)
	case cb.Data == "show_answer":
		h.cbShowAnswer(ctx, cb)
	case cb.Data == "next_turn":
		h.cbNextTurn(ctx, cb)
	case cb.Data == "finish_add":
		h.cbFinishAdd(ctx, cb)
	case cb.Data == "cancel_add":
		h.cbCancelAdd(ctx, cb)
	case cb.Data == "confirm_reset":
		h.cbConfirmReset(ctx, cb)
	case cb.Data == "cancel_reset":
		h.cbCancelReset(ctx, cb)
	case cb.Data == "confirm_delete":
		h.cbConfirmDelete(ctx, cb)
	case cb.Data == "cancel_delete":
		h.cbCancelDelete(ctx, cb)
	case strings.HasPrefix(cb.Data, "score_"):
		h.cbScoreButton(ctx, cb)
	}
}

func (h *Handler) cbScoreButton(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	// Обработка отмены
	if cb.Data == "score_cancel" {
		h.cbScoreCancel(ctx, cb)
		return
	}

	h.scoreStateMu.RLock()
	state, hasState := h.scoreState[cb.From.ID]
	h.scoreStateMu.RUnlock()

	if !hasState || !state.Waiting {
		return
	}

	// Парсим очки из callback data
	scoreStr := strings.TrimPrefix(cb.Data, "score_")
	score, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil {
		return
	}

	playerName, totalScore, ok := h.web.AddScoreToCurrentPlayer(score)
	if !ok {
		h.sendText(cb.Message.Chat.ID, "❌ Ошибка: нет активной сессии")
		h.clearScoreState(cb.From.ID)
		return
	}

	h.clearScoreState(cb.From.ID)

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(edit)

	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, fmt.Sprintf("✅ *%s* получает *%.1f* 🤑 BazuCoin!\n\nВсего: *%.1f* 🤑", playerName, score, totalScore))
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

func (h *Handler) cbScoreCancel(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.clearScoreState(cb.From.ID)

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(edit)

	h.sendText(cb.Message.Chat.ID, "❌ Ввод BazuCoin отменён")
}

func (h *Handler) handleAddState(ctx context.Context, msg *tgbotapi.Message, state *AddSituationState) {
	if !h.isAdmin(msg.From.ID) {
		return
	}

	// Если ещё нет ответа — ожидаем текст
	if state.Answer == "" {
		if msg.Text == "" {
			h.sendText(msg.Chat.ID, "Пожалуйста, введите текстовый ответ")
			return
		}
		state.Answer = msg.Text
		h.sendText(msg.Chat.ID, fmt.Sprintf("✅ Ответ сохранён: *%s*\n\nТеперь отправьте фотографии (от 1 до 5)", state.Answer))
		return
	}

	// Ожидаем фото
	if msg.Photo != nil && len(msg.Photo) > 0 {
		if len(state.Photos) >= 5 {
			h.sendText(msg.Chat.ID, "Максимум 5 фотографий. Нажмите 'Завершить добавление'")
			return
		}

		// Берём фото максимального размера
		photo := msg.Photo[len(msg.Photo)-1]
		state.Photos = append(state.Photos, photo.FileID)

		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("📷 Фото %d добавлено\n\nМожете отправить ещё или нажмите кнопку для завершения", len(state.Photos)))
		reply.ReplyMarkup = AddPhotoKeyboard()
		h.bot.Send(reply)
	}
}

func (h *Handler) cmdStart(ctx context.Context, msg *tgbotapi.Message) {
	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if err == service.ErrNoSituations {
			h.sendText(msg.Chat.ID, "😔 Нет доступных ситуаций. Попросите администратора добавить новые или сбросить игру командой /reset")
			return
		}
		log.Printf("Error starting round: %v", err)
		h.sendText(msg.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	photoMsg := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileID(photo.FileID))
	photoMsg.Caption = fmt.Sprintf("🎯 Угадайте, что это?\n\nФото %d из %d", current, total)
	photoMsg.ReplyMarkup = GameKeyboard(current < total)
	h.bot.Send(photoMsg)
}

func (h *Handler) cmdAdd(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(msg.From.ID) {
		h.sendText(msg.Chat.ID, "⛔ Эта команда доступна только администратору")
		return
	}

	h.addStateMu.Lock()
	h.addState[msg.From.ID] = &AddSituationState{Waiting: true}
	h.addStateMu.Unlock()

	reply := tgbotapi.NewMessage(msg.Chat.ID, "📝 *Добавление новой ситуации*\n\nВведите правильный ответ (что изображено на фото):")
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = CancelAddKeyboard()
	h.bot.Send(reply)
}

func (h *Handler) cmdReset(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(msg.From.ID) {
		h.sendText(msg.Chat.ID, "⛔ Эта команда доступна только администратору")
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "🔄 Вы уверены, что хотите сбросить игру?\n\nВсе ситуации снова станут доступны для игры.")
	reply.ReplyMarkup = ConfirmResetKeyboard()
	h.bot.Send(reply)
}

func (h *Handler) cmdDelete(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(msg.From.ID) {
		h.sendText(msg.Chat.ID, "⛔ Эта команда доступна только администратору")
		return
	}

	total, _, err := h.repo.GetStats(ctx)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		h.sendText(msg.Chat.ID, "Ошибка получения статистики")
		return
	}

	if total == 0 {
		h.sendText(msg.Chat.ID, "База данных уже пуста")
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("🗑️ *ВНИМАНИЕ!*\n\nВы собираетесь удалить ВСЕ данные:\n• Ситуаций: %d\n\nЭто действие необратимо!", total))
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = ConfirmDeleteKeyboard()
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
/delete — удалить ВСЕ ситуации и фото

*Как играть:*
1. Нажмите /start
2. Смотрите на фото и угадывайте ситуацию
3. Кнопка "Ещё" покажет фото с другого ракурса
4. "Правильный ответ" покажет ответ
5. "Следующий ход" — переход к новой ситуации

*BazuCoin:*
🤑 За каждый ход можно получить от 0 до 3 BazuCoin
Возможные значения: 0, 0.5, 1, 1.5, 2, 2.5, 3`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

// Callback handlers
func (h *Handler) cbMorePhoto(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	photo, err := h.game.NextPhoto(ctx)
	if err != nil {
		if err == service.ErrNoMorePhotos {
			h.sendText(cb.Message.Chat.ID, "Больше нет фотографий для этой ситуации")
			return
		}
		log.Printf("Error getting next photo: %v", err)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	photoMsg := tgbotapi.NewPhoto(cb.Message.Chat.ID, tgbotapi.FileID(photo.FileID))
	photoMsg.Caption = fmt.Sprintf("🎯 Угадайте, что это?\n\nФото %d из %d", current, total)
	photoMsg.ReplyMarkup = GameKeyboard(current < total)
	h.bot.Send(photoMsg)
}

func (h *Handler) cbShowAnswer(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	answer, err := h.game.GetAnswer(ctx)
	if err != nil {
		log.Printf("Error getting answer: %v", err)
		return
	}

	h.sendText(cb.Message.Chat.ID, fmt.Sprintf("✅ Правильный ответ:\n\n*%s*", answer))
}

func (h *Handler) cbNextTurn(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	// Завершаем текущий раунд
	if err := h.game.FinishRound(ctx); err != nil {
		log.Printf("Error finishing round: %v", err)
	}

	// Начинаем новый
	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if err == service.ErrNoSituations {
			h.sendText(cb.Message.Chat.ID, "🎉 Все ситуации сыграны! Используйте /reset для новой игры")
			return
		}
		log.Printf("Error starting new round: %v", err)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	photoMsg := tgbotapi.NewPhoto(cb.Message.Chat.ID, tgbotapi.FileID(photo.FileID))
	photoMsg.Caption = fmt.Sprintf("🎯 Угадайте, что это?\n\nФото %d из %d", current, total)
	photoMsg.ReplyMarkup = GameKeyboard(current < total)
	h.bot.Send(photoMsg)
}

func (h *Handler) cbFinishAdd(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	h.addStateMu.RLock()
	state, exists := h.addState[cb.From.ID]
	h.addStateMu.RUnlock()

	if !exists || state.Answer == "" || len(state.Photos) == 0 {
		h.sendText(cb.Message.Chat.ID, "❌ Нужно указать ответ и добавить хотя бы одно фото")
		return
	}

	// Сохраняем в базу
	err := h.repo.Create(ctx, state.Answer, state.Photos)
	if err != nil {
		log.Printf("Error saving situation: %v", err)
		h.sendText(cb.Message.Chat.ID, "Ошибка сохранения. Попробуйте ещё раз.")
		return
	}

	h.addStateMu.Lock()
	delete(h.addState, cb.From.ID)
	h.addStateMu.Unlock()

	h.sendText(cb.Message.Chat.ID, fmt.Sprintf("✅ Ситуация добавлена!\n\nОтвет: %s\nФотографий: %d", state.Answer, len(state.Photos)))
}

func (h *Handler) cbCancelAdd(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.addStateMu.Lock()
	delete(h.addState, cb.From.ID)
	h.addStateMu.Unlock()

	h.sendText(cb.Message.Chat.ID, "❌ Добавление отменено")
}

func (h *Handler) cbConfirmReset(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	if err := h.game.ResetGame(ctx); err != nil {
		log.Printf("Error resetting game: %v", err)
		h.sendText(cb.Message.Chat.ID, "Ошибка сброса игры")
		return
	}

	h.sendText(cb.Message.Chat.ID, "✅ Игра сброшена! Все ситуации снова доступны.")
}

func (h *Handler) cbCancelReset(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.sendText(cb.Message.Chat.ID, "❌ Сброс отменён")
}

func (h *Handler) cbConfirmDelete(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if !h.isAdmin(cb.From.ID) {
		return
	}

	count, err := h.repo.DeleteAll(ctx)
	if err != nil {
		log.Printf("Error deleting all: %v", err)
		h.sendText(cb.Message.Chat.ID, "Ошибка удаления данных")
		return
	}

	h.game.ResetGame(ctx)

	h.sendText(cb.Message.Chat.ID, fmt.Sprintf("✅ Удалено ситуаций: %d\n\nБаза данных очищена.", count))
}

func (h *Handler) cbCancelDelete(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.sendText(cb.Message.Chat.ID, "❌ Удаление отменено")
}

func (h *Handler) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}

func (h *Handler) isAdmin(userID int64) bool {
	return userID == h.adminID
}