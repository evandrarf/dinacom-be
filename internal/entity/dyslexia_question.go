package entity

import (
	"time"

	"gorm.io/gorm"
)

// QuestionBankTemplate - Template soal untuk generate
type QuestionBankTemplate struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	TemplateID       string         `gorm:"uniqueIndex;size:50;not null" json:"template_id"` // e.g. "e-bd-1"
	Difficulty       string         `gorm:"size:20;not null;index" json:"difficulty"`        // easy, medium, hard
	TargetLetterPair string         `gorm:"size:10;not null" json:"target_letter_pair"`      // b-d, p-q, etc
	TargetLetter     string         `gorm:"size:5;not null" json:"target_letter"`            // B, D, etc
	CorrectWord      string         `gorm:"size:100;not null" json:"correct_word"`           // BATU
	Distractors      string         `gorm:"type:text;not null" json:"distractors"`           // JSON array: ["DATU","MATU","SATU"]
	Hint             string         `gorm:"type:text" json:"hint"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (QuestionBankTemplate) TableName() string {
	return "question_bank_templates"
}

// GeneratedQuestion - Hasil generate dari Gemini (cache)
type GeneratedQuestion struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	QuestionID       string         `gorm:"uniqueIndex;size:100;not null" json:"question_id"` // hash unique
	TemplateID       string         `gorm:"size:50;not null;index" json:"template_id"`        // FK ke template
	Difficulty       string         `gorm:"size:20;not null;index" json:"difficulty"`
	QuestionText     string         `gorm:"type:text;not null" json:"question_text"` // "Pilih kata yang benar..."
	TargetLetterPair string         `gorm:"size:10" json:"target_letter_pair"`
	TargetLetter     string         `gorm:"size:5" json:"target_letter"`
	Options          string         `gorm:"type:text;not null" json:"options"`       // JSON array: ["BATU","DATU","MATU","SATU"]
	CorrectAnswer    string         `gorm:"size:100;not null" json:"correct_answer"` // BATU
	Hint             string         `gorm:"type:text" json:"hint"`
	GeneratedBy      string         `gorm:"size:20;default:gemini" json:"generated_by"` // gemini, fallback
	UsageCount       int            `gorm:"default:0" json:"usage_count"`               // berapa kali dipakai
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (GeneratedQuestion) TableName() string {
	return "generated_questions"
}

// UserAnswer - Jawaban user untuk setiap soal
type UserAnswer struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	UserID        string         `gorm:"size:100;not null;index" json:"user_id"`     // user identifier
	SessionID     string         `gorm:"size:100;not null;index" json:"session_id"`  // session test
	QuestionID    string         `gorm:"size:100;not null;index" json:"question_id"` // FK ke generated_questions
	UserAnswer    string         `gorm:"size:100;not null" json:"user_answer"`       // jawaban user
	CorrectAnswer string         `gorm:"size:100;not null" json:"correct_answer"`    // jawaban yang benar
	IsCorrect     bool           `gorm:"not null" json:"is_correct"`                 // benar/salah
	QuestionText  string         `gorm:"type:text" json:"question_text"`             // soal yang dijawab
	Difficulty    string         `gorm:"size:20;index" json:"difficulty"`            // difficulty soal
	AnsweredAt    time.Time      `gorm:"autoCreateTime" json:"answered_at"`          // waktu jawab
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (UserAnswer) TableName() string {
	return "user_answers"
}
