package handler

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/niko/citysnap-bot/internal/handler/fsm"
)

func (h *BotHandler) handleNickname(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	nickname := strings.TrimSpace(msg.Text)

	if len(nickname) < 3 || len(nickname) > 20 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Ник должен быть от 3 до 20 символов. Попробуй ещё раз:",
		})
		return
	}

	user, err := h.users.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		slog.Error("user not found", "tg_id", msg.From.ID)
		return
	}

	user.Nickname = nickname
	if err := h.users.Update(ctx, user); err != nil {
		slog.Error("update nickname failed", "error", err)
		return
	}

	h.fsm.Set(ctx, msg.From.ID, fsm.StateAwaitAge)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Отлично, " + nickname + "! Сколько тебе лет?",
	})
}

func (h *BotHandler) handleAge(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	age, err := strconv.Atoi(strings.TrimSpace(msg.Text))

	if err != nil || age < 18 || age > 100 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Введи возраст числом (от 18 до 100):",
		})
		return
	}

	user, err := h.users.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		return
	}
	user.Age = age
	if err := h.users.Update(ctx, user); err != nil {
		slog.Error("update age failed", "error", err)
		return
	}

	h.fsm.Set(ctx, msg.From.ID, fsm.StateAwaitCity)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "В каком городе живёшь?",
	})
}

func (h *BotHandler) handleCity(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	city := strings.TrimSpace(msg.Text)

	if len(city) < 2 || len(city) > 100 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Название города должно быть от 2 до 100 символов:",
		})
		return
	}

	user, err := h.users.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		return
	}
	user.City = city
	user.NormalizeCity()

	if err := h.users.Update(ctx, user); err != nil {
		slog.Error("update city failed", "error", err)
		return
	}

	h.fsm.Set(ctx, msg.From.ID, fsm.StateAwaitPhoto)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Отправь свою фотографию (как фото, не как файл):",
	})
}

func (h *BotHandler) handlePhoto(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message

	if msg.Photo == nil || len(msg.Photo) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Пожалуйста, отправь именно фото:",
		})
		return
	}

	// Берём последний (макс. разрешение)
	fileID := msg.Photo[len(msg.Photo)-1].FileID

	user, err := h.users.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		return
	}
	user.PhotoFileID = fileID

	if err := h.users.Update(ctx, user); err != nil {
		slog.Error("update photo failed", "error", err)
		return
	}

	h.fsm.Set(ctx, msg.From.ID, fsm.StateAwaitDescription)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Отлично! Расскажи пару слов о себе:",
	})
}

func (h *BotHandler) handleDescription(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	desc := strings.TrimSpace(msg.Text)

	if len(desc) > 500 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Описание не должно превышать 500 символов:",
		})
		return
	}

	user, err := h.users.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		return
	}
	user.Description = desc

	if err := h.users.Update(ctx, user); err != nil {
		slog.Error("update description failed", "error", err)
		return
	}

	h.fsm.Set(ctx, msg.From.ID, fsm.StateReady)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text: "🎉 Регистрация завершена!\n\n" +
			"Используй:\n" +
			"/profile — посмотреть свою анкету\n" +
			"/help — список всех команд",
	})

	slog.Info("user onboarded",
		"tg_id", msg.From.ID,
		"nickname", user.Nickname,
		"city", user.City,
	)
}
