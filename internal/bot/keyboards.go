package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// GameKeyboard — клавиатура во время игры
func GameKeyboard(hasMorePhotos bool) tgbotapi.InlineKeyboardMarkup {
	moreBtn := tgbotapi.NewInlineKeyboardButtonData("📷 Ещё", "more_photo")
	if !hasMorePhotos {
		moreBtn = tgbotapi.NewInlineKeyboardButtonData("📷 Ещё (нет)", "no_more")
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			moreBtn,
			tgbotapi.NewInlineKeyboardButtonData("✅ Правильный ответ", "show_answer"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Следующий ход", "next_turn"),
		),
	)
}

// AddPhotoKeyboard — клавиатура при добавлении фото
func AddPhotoKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Завершить добавление", "finish_add"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_add"),
		),
	)
}

// CancelAddKeyboard — клавиатура отмены добавления
func CancelAddKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_add"),
		),
	)
}

// ConfirmResetKeyboard — клавиатура подтверждения сброса
func ConfirmResetKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, сбросить", "confirm_reset"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_reset"),
		),
	)
}

// ConfirmDeleteKeyboard — клавиатура подтверждения удаления всех данных
func ConfirmDeleteKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚠️ Да, удалить ВСЁ", "confirm_delete"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_delete"),
		),
	)
}

// ScoreKeyboard — клавиатура для быстрого ввода BazuCoin
func ScoreKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("0", "score_0"),
			tgbotapi.NewInlineKeyboardButtonData("0.5", "score_0.5"),
			tgbotapi.NewInlineKeyboardButtonData("1", "score_1"),
			tgbotapi.NewInlineKeyboardButtonData("1.5", "score_1.5"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("2", "score_2"),
			tgbotapi.NewInlineKeyboardButtonData("2.5", "score_2.5"),
			tgbotapi.NewInlineKeyboardButtonData("3", "score_3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "score_cancel"),
		),
	)
}