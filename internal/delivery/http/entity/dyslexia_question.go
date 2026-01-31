package entity

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Phase string

const (
	PhaseEasy     Phase = "EASY"
	PhaseMedium   Phase = "MEDIUM"
	PhaseHard     Phase = "HARD"
	PhaseComplete Phase = "COMPLETE"
)

type QuestionTemplate struct {
	ID               string     `json:"id"`
	Difficulty       Difficulty `json:"difficulty"`
	TargetLetterPair string     `json:"targetLetterPair"`
	TargetLetter     string     `json:"targetLetter"`
	CorrectWord      string     `json:"correctWord"`
	Distractors      []string   `json:"distractors"`
}

type GeneratedQuestion struct {
	ID               string     `json:"id"`
	Difficulty       Difficulty `json:"difficulty"`
	QuestionText     string     `json:"questionText"`
	TargetLetterPair string     `json:"targetLetterPair"`
	TargetLetter     string     `json:"targetLetter"`
	Options          []string   `json:"options"`
	Answer           string     `json:"answer,omitempty"`
}

// Request untuk submit jawaban
type SubmitAnswerRequest struct {
	UserID     string `json:"user_id" validate:"required"`
	SessionID  string `json:"session_id" validate:"required"`
	QuestionID string `json:"question_id" validate:"required"`
	Answer     string `json:"answer" validate:"required"`
}

// Response untuk submit jawaban
type SubmitAnswerResponse struct {
	IsCorrect     bool   `json:"is_correct"`
	UserAnswer    string `json:"user_answer"`
	CorrectAnswer string `json:"correct_answer"`
	QuestionID    string `json:"question_id"`
	SessionID     string `json:"session_id"`
}

// User answer log untuk session
type UserAnswerLog struct {
	ID               uint   `json:"id"`
	QuestionID       string `json:"question_id"`
	QuestionText     string `json:"question_text"`
	UserAnswer       string `json:"user_answer"`
	CorrectAnswer    string `json:"correct_answer"`
	IsCorrect        bool   `json:"is_correct"`
	Difficulty       string `json:"difficulty"`
	TargetLetterPair string `json:"target_letter_pair,omitempty"`
	AnsweredAt       string `json:"answered_at"`
}

// Error pattern analysis
type ErrorPattern struct {
	LetterPair string `json:"letter_pair"`
	ErrorCount int    `json:"error_count"`
	TotalCount int    `json:"total_count"`
	ErrorRate  string `json:"error_rate"`
}

// Session report response
type SessionReport struct {
	SessionID       string         `json:"session_id"`
	TotalQuestions  int            `json:"total_questions"`
	CorrectAnswers  int            `json:"correct_answers"`
	WrongAnswers    int            `json:"wrong_answers"`
	AccuracyRate    string         `json:"accuracy_rate"`
	OverallValue    string         `json:"overall_value"`
	ErrorPatterns   []ErrorPattern `json:"error_patterns"`
	DifficultyStats map[string]int `json:"difficulty_stats"`
	AIAnalysys      string         `json:"ai_analysis"`
	Recommendations string         `json:"recommendations"`
}

// Chat request
type ChatRequest struct {
	Message string `json:"message" validate:"required"`
}

// Chat response
type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}

// Chat history item
type ChatHistoryItem struct {
	Role      string `json:"role"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}
