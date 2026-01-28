package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func GameKeyboard(hasMorePhotos bool) tgbotapi.InlineKeyboardMarkup {
	var firstRow []tgbotapi.InlineKeyboardButton

	if hasMorePhotos {
		firstRow = append(firstRow, tgbotapi.NewInlineKeyboardButtonData("📷 Ещё фото", "more_photo"))
	}
	firstRow = append(firstRow, tgbotapi.NewInlineKeyboardButtonData("✅ Показать ответ", "show_answer"))

	secondRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➡️ Следующая ситуация", "next_round"),
	}

	return tgbotapi.NewInlineKeyboardMarkup(firstRow, secondRow)
}

func AdminKeyboard(situationID int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Завершить добавление", "finish_add"),
		),
	)
}

func ConfirmResetKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, сбросить", "confirm_reset"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_reset"),
		),
	)
}

func ConfirmDeleteKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚠️ Да, удалить ВСЁ", "confirm_delete"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_delete"),
		),
	)
}