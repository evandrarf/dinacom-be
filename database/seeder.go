package database

import (
	"encoding/json"
	"fmt"

	oldEntity "github.com/evandrarf/dinacom-be/internal/delivery/http/entity"
	"github.com/evandrarf/dinacom-be/internal/entity"
	"gorm.io/gorm"
)

// QuestionBankData - Static data untuk seed (fokus pada kata dengan huruf mirip)
var QuestionBankData = []oldEntity.QuestionTemplate{
	// ==================== EASY QUESTIONS (kata pendek 4-5 huruf, 1 pasangan huruf mirip) ====================
	// b-d confusion

}

// SeedQuestionBank - Migrate data dari QuestionBankData ke database
func SeedQuestionBank(db *gorm.DB) error {
	// Check if already seeded
	var count int64
	db.Model(&entity.QuestionBankTemplate{}).Count(&count)
	if count > 0 {
		fmt.Println("Question bank already seeded, skipping...")
		return nil
	}

	fmt.Println("Seeding question bank templates...")

	for _, tpl := range QuestionBankData {
		// Convert distractors to JSON string
		distractorsJSON, err := json.Marshal(tpl.Distractors)
		if err != nil {
			return fmt.Errorf("failed to marshal distractors for %s: %w", tpl.ID, err)
		}

		template := entity.QuestionBankTemplate{
			TemplateID:       tpl.ID,
			Difficulty:       string(tpl.Difficulty),
			TargetLetterPair: tpl.TargetLetterPair,
			TargetLetter:     tpl.TargetLetter,
			CorrectWord:      tpl.CorrectWord,
			Distractors:      string(distractorsJSON),
		}

		if err := db.Create(&template).Error; err != nil {
			return fmt.Errorf("failed to seed template %s: %w", tpl.ID, err)
		}
	}

	fmt.Printf("Successfully seeded %d question bank templates\n", len(QuestionBankData))
	return nil
}
