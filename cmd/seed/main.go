package main

import (
	"flag"
	"log"
	"time"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/platform/database"
	"gameco-technical-assessment/internal/seeds"
)

func main() {
	mode := flag.String("mode", "demo", "seed mode: demo or production")
	flag.Parse()

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

	var runner *seeds.Runner
	switch *mode {
	case "demo":
		runner = seeds.NewRunner(db)
	case "production":
		runner = seeds.NewProductionRunner(db)
	default:
		log.Fatalf("unsupported seed mode %q: expected demo or production", *mode)
	}

	report, err := runner.Run()
	if err != nil {
		log.Fatal(err)
	}

	for _, result := range report.Results {
		log.Printf("OK   %-24s (%.2fms)", result.Name, float64(result.Duration)/float64(time.Millisecond))
	}
	log.Printf("seed command \"run\" finished")
}
