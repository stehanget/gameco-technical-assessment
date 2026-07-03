package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/platform/database"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if err := database.RunGoose(ctx, cfg, command); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("goose command %q finished\n", command)
}
