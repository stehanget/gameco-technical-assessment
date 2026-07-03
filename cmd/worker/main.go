package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"gameco-technical-assessment/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerApp, err := app.NewWorkerApp()
	if err != nil {
		log.Fatal(err)
	}

	if err := workerApp.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
