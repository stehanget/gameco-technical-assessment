package main

import (
	"fmt"
	"log"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/platform/database"
	"gameco-technical-assessment/internal/seeds"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewGORM(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = database.CloseGORM(db)
	}()

	report, err := seeds.NewRunner(db).Run()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(report.Summary())
}
