package database

import (
	"github.com/evandrarf/dinacom-be/internal/entity"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&entity.QuestionBankTemplate{},
		&entity.GeneratedQuestion{},
		&entity.UserAnswer{},
	)
	return err
}
