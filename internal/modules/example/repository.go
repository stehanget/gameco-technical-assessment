package example

import "gorm.io/gorm"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List() ([]Example, error) {
	var examples []Example
	if err := r.db.Order("id DESC").Find(&examples).Error; err != nil {
		return nil, err
	}

	return examples, nil
}

func (r *Repository) Create(example Example) (Example, error) {
	if err := r.db.Create(&example).Error; err != nil {
		return Example{}, err
	}

	return example, nil
}
