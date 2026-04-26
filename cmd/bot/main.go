package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/niko/citysnap-bot/internal/config"
	"github.com/niko/citysnap-bot/internal/handler"
	"github.com/niko/citysnap-bot/internal/handler/fsm"
	"github.com/niko/citysnap-bot/internal/repository"
	"github.com/niko/citysnap-bot/internal/repository/cache"
	"github.com/niko/citysnap-bot/internal/server"
	"github.com/niko/citysnap-bot/internal/service"
)

var version = "dev"

func main() {
	cfg := config.MustLoad()
	setupLogger(cfg.AppEnv)

	slog.Info("starting CitySnap Bot", "version", version, "env", cfg.AppEnv)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// ── PostgreSQL ──────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("postgres connected")

	// ── Redis ───────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("redis connected")

	// ── Repositories ────────────────────────────────
	userRepo := repository.NewUserRepo(pool)
	swipeRepo := repository.NewSwipeRepo(pool)
	matchRepo := repository.NewMatchRepo(pool)
	photoRepo := repository.NewDailyPhotoRepo(pool)
	photoCache := cache.NewPhotoCache(rdb)

	// ── Services ────────────────────────────────────
	userSvc := service.NewUserService(userRepo)
	swipeSvc := service.NewSwipeService(swipeRepo, matchRepo, userRepo)
	photoSvc := service.NewDailyPhotoService(photoRepo, photoCache)

	// ── FSM ─────────────────────────────────────────
	fsmStore := fsm.NewStorage(rdb)

	// ── Background workers ──────────────────────────
	go photoSvc.StartCleanupWorker(ctx)

	// ── HTTP health server ──────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", server.HealthHandler(pool, rdb))

	healthSrv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		slog.Info("health server started", "addr", ":8080")
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
		}
	}()

	// ── Telegram Bot ────────────────────────────────
	if cfg.TelegramToken == "" {
		slog.Warn("TELEGRAM_TOKEN not set, bot will not start (only health server)")
		<-ctx.Done()
		shutdown(healthSrv)
		return
	}

	botHandler := handler.New(userSvc, swipeSvc, photoSvc, fsmStore)

	b, err := bot.New(cfg.TelegramToken,
		bot.WithDefaultHandler(botHandler.DefaultHandler),
	)
	if err != nil {
		slog.Error("bot init failed", "error", err)
		os.Exit(1)
	}

	botHandler.Register(b)

	slog.Info("CitySnap Bot ready",
		"health", "http://localhost:8080/readyz",
		"bot", "running",
	)

	// b.Start(ctx) — блокирует до отмены ctx
	b.Start(ctx)

	// ── Graceful shutdown ───────────────────────────
	shutdown(healthSrv)
}

func shutdown(healthSrv *http.Server) {
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthSrv.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
}

func setupLogger(env string) {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	slog.SetDefault(slog.New(handler))
}
