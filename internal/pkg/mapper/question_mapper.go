package mapper

import (
	"encoding/json"

	oldEntity "github.com/evandrarf/dinacom-be/internal/delivery/http/entity"
	dbEntity "github.com/evandrarf/dinacom-be/internal/entity"
)

// ConvertToQuestionTemplate - Convert DB entity to domain entity
func ConvertToQuestionTemplate(dbTemplate *dbEntity.QuestionBankTemplate) (oldEntity.QuestionTemplate, error) {
	var distractors []string
	if err := json.Unmarshal([]byte(dbTemplate.Distractors), &distractors); err != nil {
		return oldEntity.QuestionTemplate{}, err
	}

	return oldEntity.QuestionTemplate{
		ID:               dbTemplate.TemplateID,
		Difficulty:       oldEntity.Difficulty(dbTemplate.Difficulty),
		TargetLetterPair: dbTemplate.TargetLetterPair,
		TargetLetter:     dbTemplate.TargetLetter,
		CorrectWord:      dbTemplate.CorrectWord,
		Distractors:      distractors,
	}, nil
}
