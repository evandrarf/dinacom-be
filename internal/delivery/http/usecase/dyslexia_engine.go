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
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type DyslexiaQuestionUsecase interface {
	Generate(ctx context.Context, difficulty entity.Difficulty, count int, includeAnswer bool, pattern string, useAI bool) ([]entity.GeneratedQuestion, error)
	SubmitAnswer(ctx context.Context, req entity.SubmitAnswerRequest) (*entity.SubmitAnswerResponse, error)
	GetSessionAnswers(ctx context.Context, sessionID string) ([]entity.UserAnswerLog, error)
	GenerateSessionReport(ctx context.Context, sessionID string) (*entity.SessionReport, error)
	ChatWithBot(ctx context.Context, sessionID string, userMessage string) (*entity.ChatResponse, error)
	GetChatHistory(ctx context.Context, sessionID string) ([]entity.ChatHistoryItem, error)
}

type DyslexiaQuestionConfig struct {
	DB             *gorm.DB
	Gemini         *llm.GeminiClient
	PromptTemplate string
	Repository     repository.DyslexiaQuestionRepository
	Config         *viper.Viper
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

func (u *dyslexiaQuestionUsecase) Generate(ctx context.Context, difficulty entity.Difficulty, count int, includeAnswer bool, pattern string, useAI bool) ([]entity.GeneratedQuestion, error) {
	startTime := time.Now()
	fmt.Printf("[PERF] Generate started for difficulty=%s count=%d pattern=%s use_ai=%v\n", difficulty, count, pattern, useAI)

	if difficulty == "" {
		difficulty = entity.DifficultyEasy
	}
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	// Define common letter pairs for dyslexia practice
	letterPairs := []string{"b-d", "p-q", "m-w", "n-u", "m-n"}

	// If pattern is specified, validate and use only that pattern
	if pattern != "" {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		validPattern := false
		for _, lp := range letterPairs {
			if lp == pattern {
				validPattern = true
				break
			}
		}
		if !validPattern {
			return nil, fmt.Errorf("invalid pattern: %s (allowed: b-d, p-q, m-w, n-u, m-n)", pattern)
		}
		letterPairs = []string{pattern} // Use only the specified pattern
	}

	// If use_ai=false, retrieve from DB cache
	if !useAI {
		fmt.Printf("[PERF] Using DB cache (use_ai=false)\n")
		return u.generateFromDBCache(ctx, difficulty, count, includeAnswer, pattern)
	}

	// Check if AI prompt is disabled via env
	disableAI := u.cfg.Config.GetBool("llm.gemini.disable_ai_prompt")

	// Use goroutines for parallel generation to speed up
	type result struct {
		question entity.GeneratedQuestion
		index    int
		err      error
	}

	resultChan := make(chan result, count)

	// Generate all questions in parallel
	for i := 0; i < count; i++ {
		go func(index int) {
			iterStart := time.Now()
			// Pick random letter pair for each question
			letterPair := letterPairs[u.rnd.Intn(len(letterPairs))]

			var q entity.GeneratedQuestion
			var err error

			if disableAI {
				// Skip AI, use simple fallback
				q = createFallbackQuestion(difficulty, letterPair, true)
			} else {
				// Generate from AI
				aiStart := time.Now()
				q, err = u.generateFromAI(ctx, difficulty, letterPair, true)
				fmt.Printf("[PERF] AI call %d took: %v\n", index+1, time.Since(aiStart))

				if err != nil {
					fmt.Printf("Question %d: AI generate error: %v, using fallback\n", index+1, err)
					q = createFallbackQuestion(difficulty, letterPair, true)
				} else {
					// Save asynchronously (non-blocking)
					go func(question entity.GeneratedQuestion, pair string) {
						if saveErr := u.saveGeneratedToDB(ctx, question, pair); saveErr != nil {
							fmt.Printf("Warning: failed to save question to DB: %v\n", saveErr)
						}
					}(q, letterPair)
				}
			}

			fmt.Printf("[PERF] Question %d took: %v\n", index+1, time.Since(iterStart))
			resultChan <- result{question: q, index: index, err: err}
		}(i)
	}

	// Collect results
	results := make([]entity.GeneratedQuestion, count)
	for i := 0; i < count; i++ {
		r := <-resultChan
		results[r.index] = r.question
	}

	// Remove answer from response if not requested by user
	if !includeAnswer {
		for i := range results {
			results[i].Answer = ""
		}
	}

	fmt.Printf("[PERF] Total Generate time: %v (parallel execution)\n", time.Since(startTime))
	return results, nil
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
		QuestionText:     "Dengarkan kata berikut: ",
		TargetLetterPair: dbQ.TargetLetterPair,
		TargetLetter:     dbQ.TargetLetter,
		Options:          options,
	}
	if includeAnswer {
		q.Answer = dbQ.CorrectAnswer
	}

	// Increment usage count
	_ = u.cfg.Repository.IncrementUsageCount(u.cfg.DB, dbQ.QuestionID)

	return q, nil
}

func (u *dyslexiaQuestionUsecase) saveGeneratedToDB(_ context.Context, q entity.GeneratedQuestion, letterPair string) error {
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
		TemplateID:       letterPair, // Use letterPair as template ID
		Difficulty:       string(q.Difficulty),
		QuestionText:     q.QuestionText,
		TargetLetterPair: q.TargetLetterPair,
		TargetLetter:     q.TargetLetter,
		Options:          string(optionsJSON),
		CorrectAnswer:    q.Answer,
		GeneratedBy:      "ai",
		UsageCount:       1,
	}

	return u.cfg.Repository.CreateGenerated(u.cfg.DB, dbQuestion)
}

// generateFromDBCache retrieves previously generated questions from database
func (u *dyslexiaQuestionUsecase) generateFromDBCache(_ context.Context, difficulty entity.Difficulty, count int, includeAnswer bool, pattern string) ([]entity.GeneratedQuestion, error) {
	startTime := time.Now()

	// Build filters for repository query
	filters := []string{}
	if pattern != "" {
		filters = append(filters, fmt.Sprintf("target_letter_pair = '%s'", pattern))
	}

	// Get random questions from DB matching criteria
	dbQuestions, err := u.cfg.Repository.FindRandomGeneratedByDifficulty(u.cfg.DB, string(difficulty), count, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve questions from cache: %w", err)
	}

	if len(dbQuestions) == 0 {
		return nil, fmt.Errorf("no cached questions found for difficulty=%s pattern=%s", difficulty, pattern)
	}

	// Convert DB questions to response format
	results := make([]entity.GeneratedQuestion, 0, len(dbQuestions))
	for _, dbQ := range dbQuestions {
		// Unmarshal options
		var options []string
		if err := json.Unmarshal([]byte(dbQ.Options), &options); err != nil {
			fmt.Printf("Warning: failed to parse options for question %s: %v\n", dbQ.QuestionID, err)
			continue
		}

		q := entity.GeneratedQuestion{
			ID:               dbQ.QuestionID,
			Difficulty:       entity.Difficulty(dbQ.Difficulty),
			QuestionText:     dbQ.QuestionText,
			TargetLetterPair: dbQ.TargetLetterPair,
			TargetLetter:     dbQ.TargetLetter,
			Options:          options,
		}
		if includeAnswer {
			q.Answer = dbQ.CorrectAnswer
		}

		results = append(results, q)

		// Increment usage count asynchronously
		go func(questionID string) {
			if err := u.cfg.Repository.IncrementUsageCount(u.cfg.DB, questionID); err != nil {
				fmt.Printf("Warning: failed to increment usage count for %s: %v\n", questionID, err)
			}
		}(dbQ.QuestionID)
	}

	fmt.Printf("[PERF] DB cache retrieval took: %v (found %d questions)\n", time.Since(startTime), len(results))
	return results, nil
}

// Simple fallback when AI is disabled or fails
func createFallbackQuestion(difficulty entity.Difficulty, letterPair string, includeAnswer bool) entity.GeneratedQuestion {
	// Hardcoded fallback examples per letter pair (natural lowercase for common nouns)
	fallbackWords := map[string][]string{
		"b-d": {"bola", "dola", "bela", "dela"},
		"p-q": {"pagi", "qagi", "patu", "qatu"},
		"m-w": {"maju", "waju", "mata", "wata"},
		"n-u": {"nasi", "uasi", "nama", "uama"},
		"m-n": {"makan", "nakan", "main", "nain"},
	}

	words, ok := fallbackWords[letterPair]
	if !ok {
		words = []string{"bola", "dola", "bela", "dela"} // Default
	}

	correctAnswer := words[0]
	id := generateQuestionID(correctAnswer, difficulty)

	q := entity.GeneratedQuestion{
		ID:               id,
		Difficulty:       difficulty,
		QuestionText:     "Dengarkan kata berikut: ",
		TargetLetterPair: letterPair,
		TargetLetter:     strings.Split(letterPair, "-")[0],
		Options:          words,
	}
	if includeAnswer {
		q.Answer = correctAnswer
	}
	return q
}

type geminiQuestionJSON struct {
	CorrectAnswer string   `json:"correctAnswer"`
	Options       []string `json:"options"`
}

type geminiBatchJSON struct {
	Questions []geminiQuestionJSON `json:"questions"`
}

// generateBatchFromAI generates multiple questions in ONE API call
func (u *dyslexiaQuestionUsecase) generateBatchFromAI(ctx context.Context, difficulty entity.Difficulty, count int, letterPairs []string, includeAnswer bool) ([]entity.GeneratedQuestion, error) {
	if u.cfg.Gemini == nil {
		return nil, fmt.Errorf("gemini client not configured")
	}

	// Build batch prompt asking for N questions at once
	pairsStr := strings.Join(letterPairs, ", ")
	prompt := fmt.Sprintf(`Generate %d different listening questions for Indonesian dyslexic children.

Difficulty: %s
Available letter pairs to use: %s

For each question:
1. Choose ONE letter pair from the list
2. Create ONE real Indonesian word containing that pair
3. Generate EXACTLY 3 UNIQUE distractor words that look visually similar (swap confusing letters)
4. ALL 4 OPTIONS MUST BE DIFFERENT - NO DUPLICATES ALLOWED
5. Use NATURAL capitalization (lowercase for common nouns, capitalize proper nouns)

Return JSON array of %d questions. Each question must have:
- "correctAnswer": the correct word to be spoken (with natural capitalization)
- "options": array of 4 UNIQUE words shuffled randomly (1 correct + 3 unique distractors)

CRITICAL: Ensure all 4 options in each question are UNIQUE and DIFFERENT!

IMPORTANT: Return ONLY valid JSON, NO markdown, NO code blocks.
JSON format:
{"questions":[{"correctAnswer":"bola","options":["bola","dola","bela","pola"]},{"correctAnswer":"kata","options":["kata","data","kaca","kapa"]},...]}`,
		count, difficulty, pairsStr, count)

	text, err := u.cfg.Gemini.GenerateText(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var parsed geminiBatchJSON
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		fmt.Printf("Batch JSON Parse Error - Raw output (%d chars): %s\n", len(clean), clean)
		return nil, fmt.Errorf("AI output is not valid json: %w", err)
	}

	if len(parsed.Questions) == 0 {
		return nil, fmt.Errorf("AI returned no questions")
	}

	// Convert to GeneratedQuestion format
	results := make([]entity.GeneratedQuestion, 0, len(parsed.Questions))
	for _, qData := range parsed.Questions {
		if len(qData.Options) < 2 {
			continue // Skip invalid questions
		}

		// Deduplicate options (in case AI returns duplicates)
		uniqueOptions := deduplicateOptions(qData.Options, qData.CorrectAnswer)
		if len(uniqueOptions) < 2 {
			continue // Skip if not enough unique options
		}

		// Detect letter pair from correct answer
		letterPair := detectLetterPair(qData.CorrectAnswer, letterPairs)
		targetLetter := strings.Split(letterPair, "-")[0]

		id := generateQuestionID(qData.CorrectAnswer, difficulty)
		q := entity.GeneratedQuestion{
			ID:               id,
			Difficulty:       difficulty,
			QuestionText:     "Dengarkan kata berikut: ",
			TargetLetterPair: letterPair,
			TargetLetter:     targetLetter,
			Options:          uniqueOptions,
		}
		if includeAnswer {
			q.Answer = qData.CorrectAnswer
		}
		results = append(results, q)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid questions generated")
	}

	return results, nil
}

// deduplicateOptions removes duplicate options and ensures correct answer is included
func deduplicateOptions(options []string, correctAnswer string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(options))

	// Ensure correct answer is first
	if correctAnswer != "" && !seen[correctAnswer] {
		unique = append(unique, correctAnswer)
		seen[correctAnswer] = true
	}

	// Add other unique options
	for _, opt := range options {
		if opt != "" && !seen[opt] {
			unique = append(unique, opt)
			seen[opt] = true
		}
	}

	return unique
}

// detectLetterPair detects which letter pair is in the word
func detectLetterPair(word string, letterPairs []string) string {
	word = strings.ToLower(word)
	for _, pair := range letterPairs {
		letters := strings.Split(pair, "-")
		if strings.Contains(word, letters[0]) || strings.Contains(word, letters[1]) {
			return pair
		}
	}
	return letterPairs[0] // Default fallback
}

func (u *dyslexiaQuestionUsecase) generateFromAI(ctx context.Context, difficulty entity.Difficulty, letterPair string, includeAnswer bool) (entity.GeneratedQuestion, error) {
	if u.cfg.Gemini == nil {
		return entity.GeneratedQuestion{}, fmt.Errorf("gemini client not configured")
	}

	prompt := u.cfg.PromptTemplate
	prompt = strings.ReplaceAll(prompt, "{{difficulty}}", string(difficulty))
	prompt = strings.ReplaceAll(prompt, "{{targetLetterPair}}", letterPair)

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
	if len(clean) < 30 {
		fmt.Printf("WARNING: AI response too short (%d chars): %s\n", len(clean), clean)
	}

	var parsed geminiQuestionJSON
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		fmt.Printf("JSON Parse Error - Raw output (%d chars): %s\n", len(clean), clean)
		return entity.GeneratedQuestion{}, fmt.Errorf("AI output is not valid json: %w", err)
	}
	if len(parsed.Options) < 2 || parsed.CorrectAnswer == "" {
		return entity.GeneratedQuestion{}, fmt.Errorf("AI output missing required fields")
	}

	// Deduplicate options (in case AI returns duplicates)
	uniqueOptions := deduplicateOptions(parsed.Options, parsed.CorrectAnswer)
	if len(uniqueOptions) < 2 {
		return entity.GeneratedQuestion{}, fmt.Errorf("not enough unique options after deduplication")
	}

	id := generateQuestionID(parsed.CorrectAnswer, difficulty)
	q := entity.GeneratedQuestion{
		ID:               id,
		Difficulty:       difficulty,
		QuestionText:     "Dengarkan kata berikut: ",
		TargetLetterPair: letterPair,
		TargetLetter:     strings.Split(letterPair, "-")[0], // First letter of pair
		Options:          uniqueOptions,
	}
	if includeAnswer {
		q.Answer = parsed.CorrectAnswer
	}

	return q, nil
}

func generateQuestionID(word string, difficulty entity.Difficulty) string {
	sum := sha256.Sum256([]byte(word + "|" + string(difficulty)))
	return "q-" + hex.EncodeToString(sum[:8])
}

const defaultPromptTemplate = `You are generating audio-based listening questions for Indonesian dyslexic children (TK-SD).

Design principles:
- The question text is ALWAYS static: "Dengarkan kata berikut: "
- This is a LISTENING test where a word will be spoken aloud
- Child must identify the spoken word from 4 visual options
- Focus on Indonesian words with confusing letter pairs that dyslexic children struggle with
- Use UPPERCASE for all options to aid visual recognition

Difficulty levels:
- EASY: Short words (4-5 letters) with ONE confusing letter pair (e.g., bola vs dola, pagi vs qagi)
- MEDIUM: Medium words (5-6 letters) with confusing letters in multiple positions (e.g., bunga vs dunga, panas vs qanas)
- HARD: Longer words (6+ letters) with multiple confusing letter patterns (e.g., beruang vs deruang, membaca vs memdaca)

Common confusing pairs: b-d, p-q, m-w, n-u, m-n

Parameters:
Difficulty: {{difficulty}}
Target letter pair: {{targetLetterPair}}

Task:
1. Choose ONE real Indonesian word that contains the target letter pair
2. Create 3 distractor words that LOOK visually similar (swap letters from confusing pairs)
3. Distractors should be visually plausible but may not be real words
4. Return 4 options shuffled randomly (1 correct + 3 distractors)
5. Also return the correct answer

IMPORTANT: Return ONLY valid JSON, NO markdown, NO code blocks.
JSON format:
{"correctAnswer":"KATA","options":["KATA","DATA","KAFA","KAFA"]}
`

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

	// Save analysis to cache for chatbot
	if err := u.saveAnalysisCache(ctx, report); err != nil {
		fmt.Printf("Warning: failed to save analysis cache: %v\n", err)
	}

	return report, nil
}

func (u *dyslexiaQuestionUsecase) saveAnalysisCache(_ context.Context, report *entity.SessionReport) error {
	// Convert error patterns and difficulty stats to JSON
	errorPatternsJSON, err := json.Marshal(report.ErrorPatterns)
	if err != nil {
		return err
	}

	difficultyStatsJSON, err := json.Marshal(report.DifficultyStats)
	if err != nil {
		return err
	}

	cache := &internalEntity.SessionAnalysisCache{
		SessionID:       report.SessionID,
		TotalQuestions:  report.TotalQuestions,
		CorrectAnswers:  report.CorrectAnswers,
		WrongAnswers:    report.WrongAnswers,
		AccuracyRate:    report.AccuracyRate,
		OverallValue:    report.OverallValue,
		AIAnalysis:      report.AIAnalysys,
		Recommendations: report.Recommendations,
		ErrorPatterns:   string(errorPatternsJSON),
		DifficultyStats: string(difficultyStatsJSON),
	}

	return u.cfg.Repository.CreateOrUpdateAnalysisCache(u.cfg.DB, cache)
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

// ChatWithBot handles chatbot conversation with session context
func (u *dyslexiaQuestionUsecase) ChatWithBot(ctx context.Context, sessionID string, userMessage string) (*entity.ChatResponse, error) {
	// 1. Check for cached analysis, generate if missing
	cachedAnalysis, err := u.cfg.Repository.FindAnalysisCacheBySessionID(u.cfg.DB, sessionID)
	if err != nil || cachedAnalysis == nil {
		// Generate report to create analysis cache
		_, err := u.GenerateSessionReport(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate analysis for chatbot: %w", err)
		}
		// Fetch again after generation
		cachedAnalysis, err = u.cfg.Repository.FindAnalysisCacheBySessionID(u.cfg.DB, sessionID)
		if err != nil || cachedAnalysis == nil {
			return nil, fmt.Errorf("failed to fetch analysis cache: %w", err)
		}
	}

	// 2. Build system context from cached analysis
	systemContext := fmt.Sprintf(`Kamu adalah asisten pembelajaran yang membantu anak-anak dengan disleksia dalam bahasa Indonesia.

Konteks Sesi Latihan:
- Total Soal: %d
- Jawaban Benar: %d
- Jawaban Salah: %d
- Tingkat Akurasi: %s
- Nilai Keseluruhan: %s

Analisis AI:
%s

Rekomendasi:
%s

Tugas kamu:
1. Berikan dukungan positif dan motivasi
2. Jawab pertanyaan anak dengan bahasa yang sederhana dan ramah
3. Berikan penjelasan tambahan tentang kesulitan yang mereka hadapi
4. Jangan memberikan jawaban langsung untuk soal, tapi berikan petunjuk
5. Gunakan emoji secara wajar untuk membuat percakapan lebih menyenangkan`,
		cachedAnalysis.TotalQuestions,
		cachedAnalysis.CorrectAnswers,
		cachedAnalysis.WrongAnswers,
		cachedAnalysis.AccuracyRate,
		cachedAnalysis.OverallValue,
		cachedAnalysis.AIAnalysis,
		cachedAnalysis.Recommendations,
	)

	// 3. Retrieve last 10 chat messages for conversation continuity
	chatHistory, err := u.cfg.Repository.FindChatMessagesBySessionID(u.cfg.DB, sessionID, 10)
	if err != nil {
		chatHistory = []internalEntity.ChatMessage{} // Continue with empty history
	}

	// 4. Build OpenAI messages array
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemContext,
		},
	}

	// Add chat history
	for _, msg := range chatHistory {
		var role string
		if msg.Role == "user" {
			role = openai.ChatMessageRoleUser
		} else {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Message,
		})
	}

	// Add current user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	// 5. Call LLM with full context (plain text response)
	botResponse, err := u.cfg.Gemini.GenerateChatResponse(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chatbot response: %w", err)
	}

	// 6. Save both user message and bot response to database
	// Save user message
	userMsg := &internalEntity.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Message:   userMessage,
	}
	if err := u.cfg.Repository.CreateChatMessage(u.cfg.DB, userMsg); err != nil {
		// Ignore save error, continue with response
	}

	// Save bot response
	botMsg := &internalEntity.ChatMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Message:   botResponse,
	}
	if err := u.cfg.Repository.CreateChatMessage(u.cfg.DB, botMsg); err != nil {
		// Ignore save error, continue with response
	}

	return &entity.ChatResponse{
		Response:  botResponse,
		SessionID: sessionID,
	}, nil
}

// GetChatHistory retrieves chat history for a session
func (u *dyslexiaQuestionUsecase) GetChatHistory(ctx context.Context, sessionID string) ([]entity.ChatHistoryItem, error) {
	messages, err := u.cfg.Repository.FindChatMessagesBySessionID(u.cfg.DB, sessionID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat history: %w", err)
	}

	history := make([]entity.ChatHistoryItem, 0, len(messages))
	for _, msg := range messages {
		history = append(history, entity.ChatHistoryItem{
			Role:      msg.Role,
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		})
	}

	return history, nil
}
