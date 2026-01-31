package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/evandrarf/dinacom-be/internal/delivery/http/entity"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/repository"
	internalEntity "github.com/evandrarf/dinacom-be/internal/entity"
	"github.com/evandrarf/dinacom-be/internal/pkg/llm"
	"github.com/evandrarf/dinacom-be/internal/pkg/mapper"
	"gorm.io/gorm"
)

type DyslexiaQuestionUsecase interface {
	Generate(ctx context.Context, difficulty entity.Difficulty, count int, includeAnswer bool) ([]entity.GeneratedQuestion, error)
	SubmitAnswer(ctx context.Context, req entity.SubmitAnswerRequest) (*entity.SubmitAnswerResponse, error)
	GetSessionAnswers(ctx context.Context, sessionID string) ([]entity.UserAnswerLog, error)
	GenerateSessionReport(ctx context.Context, sessionID string) (*entity.SessionReport, error)
}

type DyslexiaQuestionConfig struct {
	DB             *gorm.DB
	Gemini         *llm.GeminiClient
	PromptTemplate string
	Repository     repository.DyslexiaQuestionRepository
}

type dyslexiaQuestionUsecase struct {
	cfg DyslexiaQuestionConfig
	rnd *rand.Rand
}

func NewDyslexiaQuestionUsecase(cfg DyslexiaQuestionConfig) DyslexiaQuestionUsecase {
	if cfg.PromptTemplate == "" {
		cfg.PromptTemplate = defaultPromptTemplate
	}
	return &dyslexiaQuestionUsecase{
		cfg: cfg,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (u *dyslexiaQuestionUsecase) Generate(ctx context.Context, difficulty entity.Difficulty, count int, includeAnswer bool) ([]entity.GeneratedQuestion, error) {
	if difficulty == "" {
		difficulty = entity.DifficultyEasy
	}
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	// Load templates from database
	dbTemplates, err := u.cfg.Repository.FindTemplatesByDifficulty(u.cfg.DB, string(difficulty))
	if err != nil {
		return nil, fmt.Errorf("failed to load templates from database: %w", err)
	}

	if len(dbTemplates) < count {
		return nil, fmt.Errorf("not enough unique questions left for difficulty %s", difficulty)
	}

	// Convert DB templates to QuestionTemplate
	available := make([]entity.QuestionTemplate, 0, len(dbTemplates))
	for _, dbTpl := range dbTemplates {
		tpl, err := mapper.ConvertToQuestionTemplate(&dbTpl)
		if err != nil {
			fmt.Printf("Warning: failed to convert template %s: %v\n", dbTpl.TemplateID, err)
			continue
		}
		available = append(available, tpl)
	}

	result := make([]entity.GeneratedQuestion, 0, count)
	for i := 0; i < count; i++ {
		idx := u.rnd.Intn(len(available))
		tpl := available[idx]
		available = append(available[:idx], available[idx+1:]...)

		// Always generate with answer (for DB storage)
		q, err := u.generateFromTemplate(ctx, tpl, true)
		if err != nil {
			fmt.Printf("DyslexiaQuestionUsecase.Generate: gemini generate error: %v, trying DB fallback\n", err)
			// Try fallback from DB first
			dbFallback, dbErr := u.fallbackFromDB(ctx, tpl, true)
			if dbErr == nil {
				q = dbFallback
			} else {
				// Last resort: hardcoded fallback
				fmt.Printf("DB fallback also failed: %v, using hardcoded fallback\n", dbErr)
				q = fallbackQuestion(tpl, true)
			}
		} else {
			// Save successfully generated question to DB for future use
			if saveErr := u.saveGeneratedToDB(ctx, q, tpl.ID); saveErr != nil {
				fmt.Printf("Warning: failed to save generated question to DB: %v\n", saveErr)
			}
		}

		// Remove answer from response if not requested by user
		if !includeAnswer {
			q.Answer = ""
		}

		result = append(result, q)
	}

	return result, nil
}

func (u *dyslexiaQuestionUsecase) fallbackFromDB(_ context.Context, tpl entity.QuestionTemplate, includeAnswer bool) (entity.GeneratedQuestion, error) {
	// Try to find previously generated questions for this template from DB
	dbQuestions, err := u.cfg.Repository.FindRandomGeneratedByDifficulty(u.cfg.DB, string(tpl.Difficulty), 1, []string{})
	if err != nil || len(dbQuestions) == 0 {
		return entity.GeneratedQuestion{}, fmt.Errorf("no fallback questions in DB")
	}

	dbQ := dbQuestions[0]

	// Unmarshal options
	var options []string
	if err := json.Unmarshal([]byte(dbQ.Options), &options); err != nil {
		return entity.GeneratedQuestion{}, fmt.Errorf("failed to parse options: %w", err)
	}

	q := entity.GeneratedQuestion{
		ID:               dbQ.QuestionID,
		Difficulty:       entity.Difficulty(dbQ.Difficulty),
		QuestionText:     dbQ.QuestionText,
		TargetLetterPair: dbQ.TargetLetterPair,
		TargetLetter:     dbQ.TargetLetter,
		Options:          options,
		Hint:             dbQ.Hint,
	}
	if includeAnswer {
		q.Answer = dbQ.CorrectAnswer
	}

	// Increment usage count
	_ = u.cfg.Repository.IncrementUsageCount(u.cfg.DB, dbQ.QuestionID)

	return q, nil
}

func (u *dyslexiaQuestionUsecase) saveGeneratedToDB(_ context.Context, q entity.GeneratedQuestion, templateID string) error {
	// Check if already exists
	existing, _ := u.cfg.Repository.FindGeneratedByQuestionID(u.cfg.DB, q.ID)
	if existing != nil {
		// Already saved, just increment usage
		return u.cfg.Repository.IncrementUsageCount(u.cfg.DB, q.ID)
	}

	// Convert options to JSON
	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return err
	}

	dbQuestion := &internalEntity.GeneratedQuestion{
		QuestionID:       q.ID,
		TemplateID:       templateID,
		Difficulty:       string(q.Difficulty),
		QuestionText:     q.QuestionText,
		TargetLetterPair: q.TargetLetterPair,
		TargetLetter:     q.TargetLetter,
		Options:          string(optionsJSON),
		CorrectAnswer:    q.Answer,
		Hint:             q.Hint,
		GeneratedBy:      "gemini",
		UsageCount:       1,
	}

	return u.cfg.Repository.CreateGenerated(u.cfg.DB, dbQuestion)
}

func fallbackQuestion(tpl entity.QuestionTemplate, includeAnswer bool) entity.GeneratedQuestion {
	options := make([]string, 0, 1+len(tpl.Distractors))
	options = append(options, tpl.CorrectWord)
	options = append(options, tpl.Distractors...)

	q := entity.GeneratedQuestion{
		ID:               tpl.ID,
		Difficulty:       tpl.Difficulty,
		QuestionText:     fmt.Sprintf("Pilih kata yang benar: mana yang memakai huruf %s?", tpl.TargetLetter),
		TargetLetterPair: tpl.TargetLetterPair,
		TargetLetter:     tpl.TargetLetter,
		Options:          options,
		Hint:             tpl.Hint,
	}
	if includeAnswer {
		q.Answer = tpl.CorrectWord
	}
	return q
}

type geminiQuestionJSON struct {
	QuestionText string   `json:"questionText"`
	Options      []string `json:"options"`
}

func (u *dyslexiaQuestionUsecase) generateFromTemplate(ctx context.Context, tpl entity.QuestionTemplate, includeAnswer bool) (entity.GeneratedQuestion, error) {
	if u.cfg.Gemini == nil {
		return entity.GeneratedQuestion{}, fmt.Errorf("gemini client not configured")
	}

	distractors := strings.Join(tpl.Distractors, ", ")
	prompt := u.cfg.PromptTemplate
	prompt = strings.ReplaceAll(prompt, "{{difficulty}}", string(tpl.Difficulty))
	prompt = strings.ReplaceAll(prompt, "{{targetLetterPair}}", tpl.TargetLetterPair)
	prompt = strings.ReplaceAll(prompt, "{{targetLetter}}", tpl.TargetLetter)
	prompt = strings.ReplaceAll(prompt, "{{correctWord}}", tpl.CorrectWord)
	prompt = strings.ReplaceAll(prompt, "{{distractors}}", distractors)
	prompt = strings.ReplaceAll(prompt, "{{hint}}", tpl.Hint)

	text, err := u.cfg.Gemini.GenerateText(ctx, prompt)
	if err != nil {
		return entity.GeneratedQuestion{}, err
	}

	// Try parse JSON from model output (strip code fences if present)
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	// Debug log
	if len(clean) < 50 {
		fmt.Printf("WARNING: Gemini response too short (%d chars): %s\n", len(clean), clean)
	}

	var parsed geminiQuestionJSON
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		fmt.Printf("JSON Parse Error - Raw output (%d chars): %s\n", len(clean), clean)
		return entity.GeneratedQuestion{}, fmt.Errorf("gemini output is not valid json: %w", err)
	}
	if parsed.QuestionText == "" || len(parsed.Options) < 2 {
		return entity.GeneratedQuestion{}, fmt.Errorf("gemini output missing fields")
	}

	id := uniqueQuestionID(tpl.ID, parsed.QuestionText)
	q := entity.GeneratedQuestion{
		ID:               id,
		Difficulty:       tpl.Difficulty,
		QuestionText:     parsed.QuestionText,
		TargetLetterPair: tpl.TargetLetterPair,
		TargetLetter:     tpl.TargetLetter,
		Options:          parsed.Options,
		Hint:             tpl.Hint,
	}
	if includeAnswer {
		q.Answer = tpl.CorrectWord
	}

	return q, nil
}

func uniqueQuestionID(templateID string, questionText string) string {
	sum := sha256.Sum256([]byte(templateID + "|" + questionText))
	return templateID + "-" + hex.EncodeToString(sum[:8])
}

const defaultPromptTemplate = `You are generating a multiple-choice reading question for Indonesian dyslexic children (TK-SD).

Design principles:
- Keep instructions very short, friendly, and clear
- High letter contrast: use UPPERCASE words in options
- Focus on confusion pairs (e.g., b-d, p-q)
- Do not add extra distractors beyond the given ones

Difficulty: {{difficulty}}
Target letter pair: {{targetLetterPair}}
Target letter: {{targetLetter}}
Correct word: {{correctWord}}
Distractors (must use exactly these): {{distractors}}
Hint (optional): {{hint}}

Task:
Create a questionText in Indonesian, and an options array of 4 words containing the correct word and the distractors, shuffled.

IMPORTANT: Return ONLY valid JSON with NO additional text, NO markdown formatting, NO code blocks.
JSON format:
{"questionText":"...","options":["...","...","...","..."]}`

func (u *dyslexiaQuestionUsecase) SubmitAnswer(ctx context.Context, req entity.SubmitAnswerRequest) (*entity.SubmitAnswerResponse, error) {
	// Check if answer already exists for this user, session, and question
	existingAnswer, err := u.cfg.Repository.FindExistingAnswer(u.cfg.DB, req.UserID, req.SessionID, req.QuestionID)
	if err == nil && existingAnswer != nil {
		// Answer already exists, return existing answer without saving
		return &entity.SubmitAnswerResponse{
			IsCorrect:     existingAnswer.IsCorrect,
			UserAnswer:    existingAnswer.UserAnswer,
			CorrectAnswer: existingAnswer.CorrectAnswer,
			QuestionID:    existingAnswer.QuestionID,
			SessionID:     existingAnswer.SessionID,
		}, nil
	}

	// Find the generated question from database
	generatedQ, err := u.cfg.Repository.FindGeneratedByQuestionID(u.cfg.DB, req.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("question not found: %w", err)
	}

	// Normalize answers for comparison (case-insensitive, trim spaces)
	userAnswer := strings.TrimSpace(strings.ToUpper(req.Answer))
	correctAnswer := strings.TrimSpace(strings.ToUpper(generatedQ.CorrectAnswer))
	isCorrect := userAnswer == correctAnswer

	// Save to database
	userAnswerEntity := &internalEntity.UserAnswer{
		UserID:        req.UserID,
		SessionID:     req.SessionID,
		QuestionID:    req.QuestionID,
		UserAnswer:    req.Answer,
		CorrectAnswer: generatedQ.CorrectAnswer,
		IsCorrect:     isCorrect,
		QuestionText:  generatedQ.QuestionText,
		Difficulty:    generatedQ.Difficulty,
	}

	if err := u.cfg.Repository.CreateUserAnswer(u.cfg.DB, userAnswerEntity); err != nil {
		return nil, fmt.Errorf("failed to save answer: %w", err)
	}

	// Return response
	response := &entity.SubmitAnswerResponse{
		IsCorrect:     isCorrect,
		UserAnswer:    req.Answer,
		CorrectAnswer: generatedQ.CorrectAnswer,
		QuestionID:    req.QuestionID,
		SessionID:     req.SessionID,
	}

	return response, nil
}

func (u *dyslexiaQuestionUsecase) GetSessionAnswers(ctx context.Context, sessionID string) ([]entity.UserAnswerLog, error) {
	// Get all answers for this session
	answers, err := u.cfg.Repository.FindUserAnswersBySessionID(u.cfg.DB, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session answers: %w", err)
	}

	// Convert to UserAnswerLog
	logs := make([]entity.UserAnswerLog, 0, len(answers))
	for _, answer := range answers {
		// Get generated question to fetch target_letter_pair
		generatedQ, _ := u.cfg.Repository.FindGeneratedByQuestionID(u.cfg.DB, answer.QuestionID)

		targetLetterPair := ""
		if generatedQ != nil {
			targetLetterPair = generatedQ.TargetLetterPair
		}

		log := entity.UserAnswerLog{
			ID:               answer.ID,
			QuestionID:       answer.QuestionID,
			QuestionText:     answer.QuestionText,
			UserAnswer:       answer.UserAnswer,
			CorrectAnswer:    answer.CorrectAnswer,
			IsCorrect:        answer.IsCorrect,
			Difficulty:       answer.Difficulty,
			TargetLetterPair: targetLetterPair,
			AnsweredAt:       answer.AnsweredAt.Format(time.RFC3339),
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (u *dyslexiaQuestionUsecase) GenerateSessionReport(ctx context.Context, sessionID string) (*entity.SessionReport, error) {
	// Get all answers for this session
	answers, err := u.cfg.Repository.FindUserAnswersBySessionID(u.cfg.DB, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session answers: %w", err)
	}

	if len(answers) == 0 {
		return nil, fmt.Errorf("no answers found for session")
	}

	// Calculate basic stats
	totalQuestions := len(answers)
	correctAnswers := 0
	wrongAnswers := 0
	difficultyStats := make(map[string]int)
	letterPairErrors := make(map[string]struct {
		errors int
		total  int
	})

	for _, answer := range answers {
		if answer.IsCorrect {
			correctAnswers++
		} else {
			wrongAnswers++
		}

		// Count by difficulty
		difficultyStats[answer.Difficulty]++

		// Get letter pair info
		generatedQ, _ := u.cfg.Repository.FindGeneratedByQuestionID(u.cfg.DB, answer.QuestionID)
		if generatedQ != nil && generatedQ.TargetLetterPair != "" {
			pair := generatedQ.TargetLetterPair
			stats := letterPairErrors[pair]
			stats.total++
			if !answer.IsCorrect {
				stats.errors++
			}
			letterPairErrors[pair] = stats
		}
	}

	// Calculate accuracy
	accuracyRate := fmt.Sprintf("%.1f%%", float64(correctAnswers)/float64(totalQuestions)*100)

	// Build error patterns
	errorPatterns := make([]entity.ErrorPattern, 0)
	for pair, stats := range letterPairErrors {
		if stats.total > 0 {
			errorRate := fmt.Sprintf("%.1f%%", float64(stats.errors)/float64(stats.total)*100)
			errorPatterns = append(errorPatterns, entity.ErrorPattern{
				LetterPair: pair,
				ErrorCount: stats.errors,
				TotalCount: stats.total,
				ErrorRate:  errorRate,
			})
		}
	}

	// Generate Gemini analysis
	geminiAnalysis, recommendations, overallValue := u.generateAIAnalysis(ctx, answers, errorPatterns, accuracyRate)

	report := &entity.SessionReport{
		SessionID:       sessionID,
		TotalQuestions:  totalQuestions,
		CorrectAnswers:  correctAnswers,
		WrongAnswers:    wrongAnswers,
		AccuracyRate:    accuracyRate,
		OverallValue:    overallValue,
		ErrorPatterns:   errorPatterns,
		DifficultyStats: difficultyStats,
		AIAnalysys:      geminiAnalysis,
		Recommendations: recommendations,
	}

	return report, nil
}

func (u *dyslexiaQuestionUsecase) generateAIAnalysis(ctx context.Context, answers []internalEntity.UserAnswer, errorPatterns []entity.ErrorPattern, accuracyRate string) (string, string, string) {
	if u.cfg.Gemini == nil {
		return "AI analysis not available", "Practice more to improve", "good"
	}

	// Build analysis prompt
	prompt := fmt.Sprintf(`Analyze this dyslexia learning session data for an Indonesian child (TK-SD level):

Total Questions: %d
Accuracy Rate: %s
Wrong Answers: %d

Error Patterns by Letter Pairs:
`, len(answers), accuracyRate, len(answers)-countCorrect(answers))

	for _, pattern := range errorPatterns {
		prompt += fmt.Sprintf("- %s: %d errors out of %d questions (%s)\n",
			pattern.LetterPair, pattern.ErrorCount, pattern.TotalCount, pattern.ErrorRate)
	}

	prompt += `
Task:
1. Provide a brief, caring analysis in Indonesian about the child's learning patterns
2. Identify which letter pairs need most attention
3. Give 2-3 specific, actionable recommendations for improvement
4. Determine overall performance level by considering MULTIPLE factors:
   - Accuracy rate (primary factor)
   - Error patterns and consistency (which letter pairs are most problematic)
   - Error rate per letter pair (high error rate on specific pairs indicates focused difficulty)
   - Number of total questions attempted (shows engagement)
   - Pattern of improvement or consistent mistakes

Return response as JSON with three fields:
{"analysis":"...","recommendations":"...","overall_value":"..."}

For overall_value, use one of these Indonesian terms based on HOLISTIC evaluation:
- "excellent" (90-100% accuracy, minimal/no consistent error patterns, good engagement)
- "sangat baik" (80-89% accuracy, few errors, minor patterns, good progress)
- "baik" (70-79% accuracy, some error patterns, showing improvement potential)
- "cukup" (60-69% accuracy, notable error patterns, needs focused practice)
- "perlu peningkatan" (below 60% accuracy, significant error patterns, needs intensive support)

IMPORTANT: Don't judge only by accuracy percentage. A child with 75% accuracy but consistent errors on one specific letter pair might need different evaluation than one with same accuracy but random errors.

Keep the language simple, encouraging, and suitable for parents/teachers of young children.`

	text, err := u.cfg.Gemini.GenerateText(ctx, prompt)
	if err != nil {
		fmt.Printf("Gemini analysis error: %v\n", err)
		return "Sesi latihan telah selesai. Terus berlatih untuk meningkatkan kemampuan membaca.",
			"Fokus pada huruf-huruf yang masih sering tertukar.",
			"baik"
	}

	// Parse JSON response
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var result struct {
		Analysis        string `json:"analysis"`
		Recommendations string `json:"recommendations"`
		OverallValue    string `json:"overall_value"`
	}

	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		fmt.Printf("Text %s\n", text)
		fmt.Printf("Failed to parse Gemini analysis: %v\n", err)
		return "Sesi latihan telah selesai. Anak menunjukkan kemajuan yang baik.",
			"Terus berlatih secara konsisten untuk hasil yang lebih baik.",
			"baik"
	}

	return result.Analysis, result.Recommendations, result.OverallValue
}

func countCorrect(answers []internalEntity.UserAnswer) int {
	count := 0
	for _, a := range answers {
		if a.IsCorrect {
			count++
		}
	}
	return count
}
