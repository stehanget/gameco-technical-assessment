package seeds

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Seeder interface {
	Name() string
	Seed(db *gorm.DB) error
	Clear(db *gorm.DB) error
}

type Runner struct {
	db      *gorm.DB
	seeders []Seeder
}

type Report struct {
	Seeders []string
	Results []Result
}

type Result struct {
	Name     string
	Duration time.Duration
}

func NewRunner(db *gorm.DB) *Runner {
	return &Runner{db: db, seeders: defaultSeeders()}
}

func (r *Runner) Run() (Report, error) {
	report := Report{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, seeder := range r.seeders {
			startedAt := time.Now()
			if err := seeder.Seed(tx); err != nil {
				return fmt.Errorf("%s: %w", seeder.Name(), err)
			}
			report.Seeders = append(report.Seeders, seeder.Name())
			report.Results = append(report.Results, Result{
				Name:     seeder.Name(),
				Duration: time.Since(startedAt),
			})
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) Summary() string {
	return fmt.Sprintf("seeded: %v", r.Seeders)
}
