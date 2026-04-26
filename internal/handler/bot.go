package handler

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/niko/citysnap-bot/internal/handler/fsm"
	"github.com/niko/citysnap-bot/internal/service"
)

type BotHandler struct {
	users   *service.UserService
	swipes  *service.SwipeService
	photos  *service.DailyPhotoService
	fsm     *fsm.Storage
}

func New(
	users *service.UserService,
	swipes *service.SwipeService,
	photos *service.DailyPhotoService,
	fsmStore *fsm.Storage,
) *BotHandler {
	return &BotHandler{
		users:  users,
		swipes: swipes,
		photos: photos,
		fsm:    fsmStore,
	}
}

func (h *BotHandler) Register(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.HandleStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/profile", bot.MatchTypeExact, h.HandleProfile)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, h.HandleHelp)

	h.registerSwipeHandlers(b)
	h.registerPhotoHandlers(b)
}

func (h *BotHandler) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	tgID := update.Message.From.ID
	state, err := h.fsm.Get(ctx, tgID)
	if err != nil {
		slog.Error("fsm get failed", "error", err, "tg_id", tgID)
		return
	}

	switch state {
	case fsm.StateAwaitNickname:
		h.handleNickname(ctx, b, update)
	case fsm.StateAwaitAge:
		h.handleAge(ctx, b, update)
	case fsm.StateAwaitCity:
		h.handleCity(ctx, b, update)
	case fsm.StateAwaitPhoto:
		h.handlePhoto(ctx, b, update)
	case fsm.StateAwaitDescription:
		h.handleDescription(ctx, b, update)
	case fsm.StateAwaitSnapPhoto:
		h.handleSnapPhoto(ctx, b, update)
	case fsm.StateAwaitSnapCaption:
		h.handleSnapCaption(ctx, b, update)
	default:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Используй /help для списка команд.",
		})
	}
}

func (h *BotHandler) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgID := update.Message.From.ID

	user, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil {
		slog.Error("get user failed", "error", err, "tg_id", tgID)
		return
	}

	if user != nil && user.IsComplete() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "С возвращением, " + user.Nickname + "! 👋\nИспользуй /help для списка команд.",
		})
		return
	}

	// Регистрация / онбординг
	_, err = h.users.Register(ctx, tgID, "")
	if err != nil {
		slog.Error("register failed", "error", err, "tg_id", tgID)
		return
	}

	h.fsm.Set(ctx, tgID, fsm.StateAwaitNickname)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "Привет! Я CitySnap 📸\n" +
			"Бот для знакомств в твоём городе с механикой «фото дня».\n\n" +
			"Давай заполним твою анкету. Как тебя зовут?",
	})
}

func (h *BotHandler) HandleProfile(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	text := "*" + bot.EscapeMarkdown(user.Nickname) + "*, " + strconv.Itoa(user.Age) +
		"\n📍 " + bot.EscapeMarkdown(user.City) +
		"\n\n" + bot.EscapeMarkdown(user.Description)

	if user.PhotoFileID != "" {
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:    chatID,
			Photo:     &models.InputFileString{Data: user.PhotoFileID},
			Caption:   text,
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

func (h *BotHandler) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "Команды CitySnap Bot:\n\n" +
			"📋 *Профиль*\n" +
			"/start — регистрация\n" +
			"/profile — твоя анкета\n\n" +
			"💕 *Знакомства*\n" +
			"/search — смотреть анкеты\n" +
			"/matches — твои мэтчи\n\n" +
			"📸 *Фото дня*\n" +
			"/snap — загрузить фото на 24ч\n" +
			"/feed — фото города\n" +
			"/mysnap — твоё активное фото\n\n" +
			"/help — это сообщение",
		ParseMode: models.ParseModeMarkdown,
	})
}
