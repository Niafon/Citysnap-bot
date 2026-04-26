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

	"github.com/niko/citysnap-bot/internal/apperror"
	"github.com/niko/citysnap-bot/internal/handler/fsm"
	appmodel "github.com/niko/citysnap-bot/internal/model"
)

func (h *BotHandler) registerPhotoHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/snap", bot.MatchTypeExact, h.HandleSnap)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/feed", bot.MatchTypeExact, h.HandleFeed)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/mysnap", bot.MatchTypeExact, h.HandleMySnap)

	b.RegisterHandlerMatchFunc(
		func(update *models.Update) bool {
			return update.CallbackQuery != nil &&
				strings.HasPrefix(update.CallbackQuery.Data, "feed:")
		},
		h.HandleFeedCallback,
	)
	b.RegisterHandlerMatchFunc(
		func(update *models.Update) bool {
			return update.CallbackQuery != nil &&
				update.CallbackQuery.Data == "snap:no_caption"
		},
		h.HandleSnapNoCaption,
	)
}

func (h *BotHandler) HandleSnap(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil || !user.IsComplete() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала заполни анкету через /start",
		})
		return
	}

	h.fsm.Set(ctx, tgID, fsm.StateAwaitSnapPhoto)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "📸 Отправь фото — его увидят все в твоём городе на 24 часа.",
	})
}

func (h *BotHandler) handleSnapPhoto(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	tgID := msg.From.ID

	if msg.Photo == nil || len(msg.Photo) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Нужно именно фото!",
		})
		return
	}

	fileID := msg.Photo[len(msg.Photo)-1].FileID

	if err := h.fsm.SetData(ctx, tgID, "snap_file_id", fileID); err != nil {
		slog.Error("fsm setdata failed", "error", err)
		return
	}

	h.fsm.Set(ctx, tgID, fsm.StateAwaitSnapCaption)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Добавь подпись или нажми «Без подписи»:",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Без подписи", CallbackData: "snap:no_caption"}},
			},
		},
	})
}

func (h *BotHandler) handleSnapCaption(ctx context.Context, b *bot.Bot, update *models.Update) {
	caption := strings.TrimSpace(update.Message.Text)
	if len(caption) > 500 {
		caption = caption[:500]
	}
	h.finalizeSnap(ctx, b, update.Message.Chat.ID, update.Message.From.ID, caption)
}

func (h *BotHandler) HandleSnapNoCaption(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
	h.finalizeSnap(ctx, b, cb.Message.Message.Chat.ID, cb.From.ID, "")
}

func (h *BotHandler) finalizeSnap(ctx context.Context, b *bot.Bot, chatID, tgID int64, caption string) {
	fileID, err := h.fsm.GetData(ctx, tgID, "snap_file_id")
	if err != nil || fileID == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Что-то пошло не так. Попробуй /snap ещё раз.",
		})
		return
	}

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil {
		return
	}

	photo, err := h.photos.Create(ctx, user.ID, user.City, fileID, caption)
	if err != nil {
		if errors.Is(err, apperror.ErrSnapActive) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "У тебя уже есть активное фото! Дождись пока оно исчезнет или удали через /mysnap.",
			})
			return
		}
		slog.Error("create snap failed", "error", err)
		return
	}

	h.fsm.Clear(ctx, tgID)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf("✅ Фото опубликовано!\n\n"+
			"Его увидят все в городе %s в течение 24 часов.\n"+
			"Используй /mysnap чтобы посмотреть статистику.", user.City),
	})

	slog.Info("snap created", "user_id", user.ID, "photo_id", photo.ID)
}

func (h *BotHandler) HandleFeed(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil || !user.IsComplete() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала заполни анкету через /start",
		})
		return
	}

	feed, err := h.photos.GetCityFeed(ctx, user.City, user.ID)
	if err != nil {
		slog.Error("get feed failed", "error", err)
		return
	}

	if len(feed) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "В твоём городе пока никто не загрузил фото дня 📸\nБудь первым через /snap!",
		})
		return
	}

	h.showFeedPhoto(ctx, b, chatID, feed, 0)
}

func (h *BotHandler) HandleFeedCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	parts := strings.SplitN(cb.Data, ":", 3)
	if len(parts) != 3 {
		return
	}

	user, _ := h.users.GetByTelegramID(ctx, cb.From.ID)
	if user == nil {
		return
	}

	feed, err := h.photos.GetCityFeed(ctx, user.City, user.ID)
	if err != nil || len(feed) == 0 {
		return
	}

	idx, _ := strconv.Atoi(parts[2])

	switch parts[1] {
	case "next":
		idx = (idx + 1) % len(feed)
	case "prev":
		idx = (idx - 1 + len(feed)) % len(feed)
	case "noop":
		return
	}

	h.showFeedPhoto(ctx, b, cb.Message.Message.Chat.ID, feed, idx)
}

func (h *BotHandler) showFeedPhoto(ctx context.Context, b *bot.Bot, chatID int64, feed []appmodel.DailyPhoto, idx int) {
	if idx < 0 || idx >= len(feed) {
		return
	}
	photo := feed[idx]

	author, _ := h.users.GetByID(ctx, photo.UserID)
	authorName := "unknown"
	if author != nil {
		authorName = author.Nickname
	}

	hoursLeft := int(photo.TimeLeft().Hours())
	caption := fmt.Sprintf("@%s | 📍 %s\n⏳ осталось %dч | 👁 %d",
		bot.EscapeMarkdown(authorName),
		bot.EscapeMarkdown(photo.City),
		hoursLeft,
		photo.ViewCount,
	)
	if photo.Caption != "" {
		caption += "\n\n" + bot.EscapeMarkdown(photo.Caption)
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "◀", CallbackData: fmt.Sprintf("feed:prev:%d", idx)},
				{Text: fmt.Sprintf("%d/%d", idx+1, len(feed)), CallbackData: "feed:noop:0"},
				{Text: "▶", CallbackData: fmt.Sprintf("feed:next:%d", idx)},
			},
		},
	}

	b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Photo:       &models.InputFileString{Data: photo.PhotoFileID},
		Caption:     caption,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboard,
	})

	go func() {
		_ = h.photos.IncrementViews(context.Background(), photo.ID)
	}()
}

func (h *BotHandler) HandleMySnap(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil || user == nil {
		return
	}

	photo, err := h.photos.GetActiveByUser(ctx, user.ID)
	if err != nil {
		slog.Error("get active snap failed", "error", err)
		return
	}

	if photo == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "У тебя нет активного фото. Загрузи через /snap!",
		})
		return
	}

	hoursLeft := int(photo.TimeLeft().Hours())
	caption := fmt.Sprintf("Твоё активное фото\n\n⏳ осталось %dч | 👁 %d просмотров",
		hoursLeft, photo.ViewCount)
	if photo.Caption != "" {
		caption += "\n\n" + bot.EscapeMarkdown(photo.Caption)
	}

	b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:    chatID,
		Photo:     &models.InputFileString{Data: photo.PhotoFileID},
		Caption:   caption,
		ParseMode: models.ParseModeMarkdown,
	})
}
