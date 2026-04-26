package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"

	"github.com/niko/citysnap-bot/internal/apperror"
	"github.com/niko/citysnap-bot/internal/model"
)

func (h *BotHandler) registerSwipeHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/search", bot.MatchTypeExact, h.HandleSearch)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/matches", bot.MatchTypeExact, h.HandleMatches)
	b.RegisterHandlerMatchFunc(
		func(update *models.Update) bool {
			return update.CallbackQuery != nil &&
				strings.HasPrefix(update.CallbackQuery.Data, "swipe:")
		},
		h.HandleSwipeCallback,
	)
}

func (h *BotHandler) HandleSearch(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала зарегистрируйся через /start",
		})
		return
	}
	if !user.IsComplete() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала заполни анкету полностью. Используй /start",
		})
		return
	}

	h.showNextCandidate(ctx, b, chatID, user.ID, user.City)
}

func (h *BotHandler) HandleSwipeCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	parts := strings.SplitN(cb.Data, ":", 3)
	if len(parts) != 3 {
		return
	}
	swipeType := parts[1]
	targetID, err := uuid.Parse(parts[2])
	if err != nil {
		return
	}

	// Acknowledge callback (убираем спиннер)
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
	})

	swiper, err := h.users.GetByTelegramID(ctx, cb.From.ID)
	if err != nil || swiper == nil {
		return
	}

	match, err := h.swipes.Swipe(ctx, swiper.ID, targetID, swipeType)
	if err != nil {
		if errors.Is(err, apperror.ErrAlreadySwiped) {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ты уже оценивал эту анкету",
			})
		} else {
			slog.Error("swipe failed", "error", err, "swiper", swiper.ID)
		}
		// Всё равно показываем следующую анкету
		h.showNextCandidate(ctx, b, cb.Message.Message.Chat.ID, swiper.ID, swiper.City)
		return
	}

	// Если есть мэтч — уведомляем обоих
	if match != nil {
		target, _ := h.users.GetByID(ctx, targetID)
		if target != nil {
			h.notifyMatch(ctx, b, swiper, target)
		}
	}

	// Показываем следующую анкету
	h.showNextCandidate(ctx, b, cb.Message.Message.Chat.ID, swiper.ID, swiper.City)
}

func (h *BotHandler) showNextCandidate(ctx context.Context, b *bot.Bot, chatID int64, userID uuid.UUID, city string) {
	candidates, err := h.swipes.GetCandidates(ctx, userID, city, 1)
	if err != nil {
		slog.Error("get candidates failed", "error", err)
		return
	}

	if len(candidates) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "В твоём городе пока нет новых анкет 😔\nПригласи друзей или загляни позже.",
		})
		return
	}

	candidate := candidates[0]
	caption := fmt.Sprintf("*%s*, %d\n📍 %s\n\n%s",
		bot.EscapeMarkdown(candidate.Nickname),
		candidate.Age,
		bot.EscapeMarkdown(candidate.City),
		bot.EscapeMarkdown(candidate.Description),
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "👎", CallbackData: "swipe:dislike:" + candidate.ID.String()},
				{Text: "❤️", CallbackData: "swipe:like:" + candidate.ID.String()},
				{Text: "⭐", CallbackData: "swipe:superlike:" + candidate.ID.String()},
			},
		},
	}

	if candidate.PhotoFileID != "" {
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      chatID,
			Photo:       &models.InputFileString{Data: candidate.PhotoFileID},
			Caption:     caption,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        caption,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
	}
}

func (h *BotHandler) notifyMatch(ctx context.Context, b *bot.Bot, swiper, target *model.User) {
	swiperText := fmt.Sprintf("🎉 У вас мэтч с *%s*!\n\nНапиши первым 👉",
		bot.EscapeMarkdown(target.Nickname))
	targetText := fmt.Sprintf("🎉 У вас мэтч с *%s*!\n\nНапиши первым 👉",
		bot.EscapeMarkdown(swiper.Nickname))

	swiperKb := matchKeyboard(target.TelegramID, target.Nickname)
	targetKb := matchKeyboard(swiper.TelegramID, swiper.Nickname)

	if target.PhotoFileID != "" {
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      swiper.TelegramID,
			Photo:       &models.InputFileString{Data: target.PhotoFileID},
			Caption:     swiperText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: swiperKb,
		})
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      swiper.TelegramID,
			Text:        swiperText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: swiperKb,
		})
	}

	if swiper.PhotoFileID != "" {
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      target.TelegramID,
			Photo:       &models.InputFileString{Data: swiper.PhotoFileID},
			Caption:     targetText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: targetKb,
		})
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      target.TelegramID,
			Text:        targetText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: targetKb,
		})
	}

	slog.Info("match notification sent", "user1", swiper.ID, "user2", target.ID)
}

func matchKeyboard(targetTgID int64, nickname string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text: "💬 Написать " + nickname,
					URL:  "tg://user?id=" + strconv.FormatInt(targetTgID, 10),
				},
			},
		},
	}
}

func (h *BotHandler) HandleMatches(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: "Сначала /start"})
		return
	}

	matches, err := h.swipes.GetMatches(ctx, user.ID)
	if err != nil {
		slog.Error("get matches failed", "error", err)
		return
	}

	if len(matches) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Пока нет мэтчей. Используй /search для поиска! 🔍",
		})
		return
	}

	text := fmt.Sprintf("Твои мэтчи (%d):\n\n", len(matches))
	for i, m := range matches {
		other := m.User1ID
		if other == user.ID {
			other = m.User2ID
		}
		otherUser, _ := h.users.GetByID(ctx, other)
		if otherUser != nil {
			text += fmt.Sprintf("%d. *%s*, %d — 📍 %s\n",
				i+1,
				bot.EscapeMarkdown(otherUser.Nickname),
				otherUser.Age,
				bot.EscapeMarkdown(otherUser.City),
			)
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}
