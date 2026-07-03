package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	_ "gameco-technical-assessment/docs/swagger"
	"gameco-technical-assessment/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	apiApp, err := app.NewAPIApp(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := apiApp.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
