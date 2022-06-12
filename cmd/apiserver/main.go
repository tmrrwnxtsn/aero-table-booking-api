package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/tmrrwnxtsn/aero-table-booking-api/internal/config"
	"github.com/tmrrwnxtsn/aero-table-booking-api/internal/handler"
	"github.com/tmrrwnxtsn/aero-table-booking-api/internal/server"
	"github.com/tmrrwnxtsn/aero-table-booking-api/internal/service"
	"github.com/tmrrwnxtsn/aero-table-booking-api/internal/store/postgres"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var flagConfig = flag.String("config", "./configs/default.yml", "path to config file")

func main() {
	flag.Parse()

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.Load(*flagConfig)
	if err != nil {
		logger.Fatalf("failed to load config data: %s", err)
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatalf("failed to set logging level: %s", err)
	}
	logger.SetLevel(level)

	db, err := postgres.NewDB(cfg.DSN)
	if err != nil {
		logger.Fatalf("failed to establish database connection: %s", err)
	}

	st := postgres.NewStore(db)
	services := service.NewServices(st)
	router := handler.NewHandler(services, logger)
	srv := server.NewServer(cfg.BindAddr, router.InitRoutes())

	// серверный контекст
	srvCtx, srvStopCtx := context.WithCancel(context.Background())

	// прослушивание системных вызовов для прерывания или завершения процесса
	osSigCh := make(chan os.Signal)
	signal.Notify(osSigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		// как только придёт системный вызов, начинаем процесс завершения приложения
		<-osSigCh
		logger.Info("server shutting down gracefully...")

		// контекст для завершения работы сервера с таймаутом в 15 секунд
		shutdownCtx, shutdownStopCtx := context.WithTimeout(srvCtx, 15*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Fatal("graceful shutdown timed out... forcing exit")
			}
		}()

		if err = db.Close(); err != nil {
			logger.Fatalf("failed to close the database connection: %s", err)
		}

		// вызов метода завершения работы сервера
		if err = srv.Shutdown(shutdownCtx); err != nil {
			logger.Fatalf("server shutdown failed: %s", err)
		}

		shutdownStopCtx()
		srvStopCtx()
	}()

	// запуск сервера
	go func() {
		if err = srv.Run(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("error occurred while running server: %s", err)
		}
	}()
	logger.Infof("server is running at %v", cfg.BindAddr)

	// ожидание остановки контекста сервера
	<-srvCtx.Done()

	logger.Info("server exited gracefully")
}