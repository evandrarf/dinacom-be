package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evandrarf/dinacom-be/database"
	"github.com/evandrarf/dinacom-be/internal/config"
	"github.com/evandrarf/dinacom-be/internal/pkg/validate"
)

func main() {
	viperConfig := config.NewViper()

	log := config.NewLogger(viperConfig)
	db := database.New(viperConfig)
	validator := validate.NewValidator()
	api := config.NewAPI(viperConfig, log)

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Info("Migrations completed successfully")

	// Run seeders
	if err := database.SeedQuestionBank(db); err != nil {
		log.Fatalf("Failed to seed question bank: %v", err)
	}
	log.Info("Seeders completed successfully")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	config.Bootstrap(&config.BootstrapConfig{
		Config:    viperConfig,
		Log:       log,
		Api:       api,
		Validator: validator,
		DB:        db,
	})

	listenAddr := ":8080"

	go func() {
		if err := api.Listen(listenAddr); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := api.ShutdownWithContext(shutdownCtx); err != nil {
		log.Errorf("API shutdown error: %v", err)
	}

	log.Info("Shutting down server...")

}
