package repository

import (
	"github.com/evandrarf/dinacom-be/internal/entity"
	"gorm.io/gorm"
)

type (
	DyslexiaQuestionRepository interface {
		// Template operations
		CreateTemplate(db *gorm.DB, template *entity.QuestionBankTemplate) error
		FindTemplatesByDifficulty(db *gorm.DB, difficulty string) ([]entity.QuestionBankTemplate, error)
		FindTemplateByTemplateID(db *gorm.DB, templateID string) (*entity.QuestionBankTemplate, error)
		CountTemplatesByDifficulty(db *gorm.DB, difficulty string) (int64, error)

		// Generated question operations
		CreateGenerated(db *gorm.DB, question *entity.GeneratedQuestion) error
		FindGeneratedByQuestionID(db *gorm.DB, questionID string) (*entity.GeneratedQuestion, error)
		FindRandomGeneratedByDifficulty(db *gorm.DB, difficulty string, limit int, excludeIDs []string) ([]entity.GeneratedQuestion, error)
		IncrementUsageCount(db *gorm.DB, questionID string) error

		// User answer operations
		CreateUserAnswer(db *gorm.DB, answer *entity.UserAnswer) error
		FindUserAnswersBySessionID(db *gorm.DB, sessionID string) ([]entity.UserAnswer, error)
		FindUserAnswersByUserID(db *gorm.DB, userID string) ([]entity.UserAnswer, error)
		FindExistingAnswer(db *gorm.DB, userID, sessionID, questionID string) (*entity.UserAnswer, error)

		// Session analysis cache operations
		CreateOrUpdateAnalysisCache(db *gorm.DB, cache *entity.SessionAnalysisCache) error
		FindAnalysisCacheBySessionID(db *gorm.DB, sessionID string) (*entity.SessionAnalysisCache, error)

		// Chat message operations
		CreateChatMessage(db *gorm.DB, message *entity.ChatMessage) error
		FindChatMessagesBySessionID(db *gorm.DB, sessionID string, limit int) ([]entity.ChatMessage, error)
	}

	dyslexiaQuestionRepository struct {
		db *gorm.DB
	}
)

func NewDyslexiaQuestionRepository(db *gorm.DB) DyslexiaQuestionRepository {
	return &dyslexiaQuestionRepository{db: db}
}

// Template operations
func (r *dyslexiaQuestionRepository) CreateTemplate(db *gorm.DB, template *entity.QuestionBankTemplate) error {
	if db == nil {
		db = r.db
	}
	return db.Create(template).Error
}

func (r *dyslexiaQuestionRepository) FindTemplatesByDifficulty(db *gorm.DB, difficulty string) ([]entity.QuestionBankTemplate, error) {
	if db == nil {
		db = r.db
	}
	var templates []entity.QuestionBankTemplate
	err := db.Where("difficulty = ?", difficulty).Find(&templates).Error
	return templates, err
}

func (r *dyslexiaQuestionRepository) FindTemplateByTemplateID(db *gorm.DB, templateID string) (*entity.QuestionBankTemplate, error) {
	if db == nil {
		db = r.db
	}
	var template entity.QuestionBankTemplate
	err := db.Where("template_id = ?", templateID).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *dyslexiaQuestionRepository) CountTemplatesByDifficulty(db *gorm.DB, difficulty string) (int64, error) {
	if db == nil {
		db = r.db
	}
	var count int64
	err := db.Model(&entity.QuestionBankTemplate{}).Where("difficulty = ?", difficulty).Count(&count).Error
	return count, err
}

// Generated question operations
func (r *dyslexiaQuestionRepository) CreateGenerated(db *gorm.DB, question *entity.GeneratedQuestion) error {
	if db == nil {
		db = r.db
	}
	return db.Create(question).Error
}

func (r *dyslexiaQuestionRepository) FindGeneratedByQuestionID(db *gorm.DB, questionID string) (*entity.GeneratedQuestion, error) {
	if db == nil {
		db = r.db
	}
	var question entity.GeneratedQuestion
	err := db.Where("question_id = ?", questionID).First(&question).Error
	if err != nil {
		return nil, err
	}
	return &question, nil
}

func (r *dyslexiaQuestionRepository) FindRandomGeneratedByDifficulty(db *gorm.DB, difficulty string, limit int, excludeIDs []string) ([]entity.GeneratedQuestion, error) {
	if db == nil {
		db = r.db
	}
	var questions []entity.GeneratedQuestion
	query := db.Where("difficulty = ?", difficulty)
	if len(excludeIDs) > 0 {
		query = query.Where("question_id NOT IN ?", excludeIDs)
	}
	err := query.Order("RANDOM()").Limit(limit).Find(&questions).Error
	return questions, err
}

func (r *dyslexiaQuestionRepository) IncrementUsageCount(db *gorm.DB, questionID string) error {
	if db == nil {
		db = r.db
	}
	return db.Model(&entity.GeneratedQuestion{}).
		Where("question_id = ?", questionID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// User answer operations
func (r *dyslexiaQuestionRepository) CreateUserAnswer(db *gorm.DB, answer *entity.UserAnswer) error {
	if db == nil {
		db = r.db
	}
	return db.Create(answer).Error
}

func (r *dyslexiaQuestionRepository) FindUserAnswersBySessionID(db *gorm.DB, sessionID string) ([]entity.UserAnswer, error) {
	if db == nil {
		db = r.db
	}
	var answers []entity.UserAnswer
	err := db.Where("session_id = ?", sessionID).Order("answered_at DESC").Find(&answers).Error
	return answers, err
}

func (r *dyslexiaQuestionRepository) FindUserAnswersByUserID(db *gorm.DB, userID string) ([]entity.UserAnswer, error) {
	if db == nil {
		db = r.db
	}
	var answers []entity.UserAnswer
	err := db.Where("user_id = ?", userID).Order("answered_at DESC").Find(&answers).Error
	return answers, err
}

func (r *dyslexiaQuestionRepository) FindExistingAnswer(db *gorm.DB, userID, sessionID, questionID string) (*entity.UserAnswer, error) {
	if db == nil {
		db = r.db
	}
	var answer entity.UserAnswer
	err := db.Where("user_id = ? AND session_id = ? AND question_id = ?", userID, sessionID, questionID).First(&answer).Error
	if err != nil {
		return nil, err
	}
	return &answer, nil
}

// Session analysis cache operations
func (r *dyslexiaQuestionRepository) CreateOrUpdateAnalysisCache(db *gorm.DB, cache *entity.SessionAnalysisCache) error {
	if db == nil {
		db = r.db
	}
	// Upsert: update if exists, create if not
	return db.Where("session_id = ?", cache.SessionID).Assign(cache).FirstOrCreate(cache).Error
}

func (r *dyslexiaQuestionRepository) FindAnalysisCacheBySessionID(db *gorm.DB, sessionID string) (*entity.SessionAnalysisCache, error) {
	if db == nil {
		db = r.db
	}
	var cache entity.SessionAnalysisCache
	err := db.Where("session_id = ?", sessionID).First(&cache).Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

// Chat message operations
func (r *dyslexiaQuestionRepository) CreateChatMessage(db *gorm.DB, message *entity.ChatMessage) error {
	if db == nil {
		db = r.db
	}
	return db.Create(message).Error
}

func (r *dyslexiaQuestionRepository) FindChatMessagesBySessionID(db *gorm.DB, sessionID string, limit int) ([]entity.ChatMessage, error) {
	if db == nil {
		db = r.db
	}
	var messages []entity.ChatMessage
	query := db.Where("session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	return messages, err
}
