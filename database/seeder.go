package database

import (
	"encoding/json"
	"fmt"

	oldEntity "github.com/evandrarf/dinacom-be/internal/delivery/http/entity"
	"github.com/evandrarf/dinacom-be/internal/entity"
	"gorm.io/gorm"
)

// QuestionBankData - Static data untuk seed (copy dari dyslexia_question_bank.go)
var QuestionBankData = []oldEntity.QuestionTemplate{
	// ==================== EASY QUESTIONS (15 templates) ====================
	{ID: "e-bd-1", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BATU", Distractors: []string{"DATU", "MATU", "SATU"}, Hint: "Kata dimulai dengan huruf B, seperti BOLA"},
	{ID: "e-bd-2", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DASI", Distractors: []string{"BASI", "PASI", "NASI"}, Hint: "Kata dimulai dengan huruf D, seperti DADU"},
	{ID: "e-bd-3", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BOLA", Distractors: []string{"DOLA", "KOLA", "SOLA"}, Hint: "Kata dimulai dengan huruf B, benda bundar untuk main"},
	{ID: "e-bd-4", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DADU", Distractors: []string{"BADU", "RADU", "KADU"}, Hint: "Kata dimulai dengan huruf D, mainan kotak untuk dilempar"},
	{ID: "e-bd-5", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BUKU", Distractors: []string{"DUKU", "SUKU", "TUKU"}, Hint: "Kata dimulai dengan huruf B, untuk dibaca"},
	{ID: "e-bd-6", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BABI", Distractors: []string{"DABI", "KABI", "RABI"}, Hint: "Kata dimulai dengan huruf B, hewan berkaki empat"},
	{ID: "e-bd-7", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DADA", Distractors: []string{"BADA", "RADA", "KADA"}, Hint: "Kata dimulai dengan huruf D, bagian tubuh di depan"},
	{ID: "e-mw-1", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MAMA", Distractors: []string{"WAMA", "PAPA", "RAMA"}, Hint: "Kata dimulai dengan huruf M, sebutan untuk ibu"},
	{ID: "e-mw-2", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WAJA", Distractors: []string{"MAJA", "RAJA", "TAJA"}, Hint: "Kata dimulai dengan huruf W, bagian depan mobil"},
	{ID: "e-mw-3", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MEJA", Distractors: []string{"WEJA", "REJA", "TEJA"}, Hint: "Kata dimulai dengan huruf M, tempat makan atau belajar"},
	{ID: "e-mw-4", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WALI", Distractors: []string{"MALI", "BALI", "KALI"}, Hint: "Kata dimulai dengan huruf W, orang yang menjaga"},
	{ID: "e-pq-1", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PAKU", Distractors: []string{"QAKU", "BAKU", "MAKU"}, Hint: "Kata dimulai dengan huruf P, benda runcing dari besi"},
	{ID: "e-pq-2", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PAGI", Distractors: []string{"QAGI", "BAGI", "LAGI"}, Hint: "Kata dimulai dengan huruf P, waktu setelah bangun tidur"},
	{ID: "e-nu-1", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NASI", Distractors: []string{"UASI", "BASI", "RASI"}, Hint: "Kata dimulai dengan huruf N, makanan pokok dari beras"},
	{ID: "e-nu-2", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NAGA", Distractors: []string{"UAGA", "RAGA", "TAGA"}, Hint: "Kata dimulai dengan huruf N, hewan mitos yang besar"},
	{ID: "e-nu-3", Difficulty: oldEntity.DifficultyEasy, TargetLetterPair: "n-u", TargetLetter: "U", CorrectWord: "ULAR", Distractors: []string{"NLAR", "ILAR", "JLAR"}, Hint: "Kata dimulai dengan huruf U, hewan merayap panjang"},
	// Medium (abbreviated for brevity - add all 14)
	{ID: "m-bd-1", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BARU", Distractors: []string{"DARU", "BIRU", "DURI"}, Hint: "Kata dengan huruf B, lawan dari lama"},
	{ID: "m-bd-2", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DURI", Distractors: []string{"BURI", "BIRU", "KURI"}, Hint: "Kata dengan huruf D, benda tajam di tumbuhan"},
	{ID: "m-bd-3", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BAYI", Distractors: []string{"DAYI", "RABI", "KADI"}, Hint: "Kata dengan huruf B, anak yang baru lahir"},
	{ID: "m-bd-4", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DARI", Distractors: []string{"BARI", "HARI", "LARI"}, Hint: "Kata dengan huruf D, menunjukkan asal"},
	{ID: "m-bd-5", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BUDI", Distractors: []string{"DUDI", "RUDI", "SUDI"}, Hint: "Kata dengan huruf B, nama orang atau perilaku baik"},
	{ID: "m-bd-6", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DUIT", Distractors: []string{"BUIT", "SUIT", "TUIT"}, Hint: "Kata dengan huruf D, uang untuk belanja"},
	{ID: "m-mw-1", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MATI", Distractors: []string{"WATI", "PATI", "SATI"}, Hint: "Kata dengan huruf M, lawan dari hidup"},
	{ID: "m-mw-2", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WARNA", Distractors: []string{"MARNA", "BARNA", "DARNA"}, Hint: "Kata dengan huruf W, merah, biru, hijau adalah..."},
	{ID: "m-mw-3", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MADU", Distractors: []string{"WADU", "RADU", "PADU"}, Hint: "Kata dengan huruf M, cairan manis dari lebah"},
	{ID: "m-mw-4", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WAKTU", Distractors: []string{"MAKTU", "FAKTU", "PAKTU"}, Hint: "Kata dengan huruf W, jam menunjukkan..."},
	{ID: "m-pq-1", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PADI", Distractors: []string{"QADI", "RADI", "BADI"}, Hint: "Kata dengan huruf P, tanaman yang jadi nasi"},
	{ID: "m-pq-2", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PETA", Distractors: []string{"QETA", "META", "BETA"}, Hint: "Kata dengan huruf P, gambar wilayah atau jalan"},
	{ID: "m-nu-1", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NAMA", Distractors: []string{"UAMA", "RAMA", "TAMA"}, Hint: "Kata dengan huruf N, identitas seseorang"},
	{ID: "m-nu-2", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NANTI", Distractors: []string{"UANTI", "BANTI", "PANTI"}, Hint: "Kata dengan huruf N, menunjukkan waktu yang akan datang"},
	{ID: "m-nu-3", Difficulty: oldEntity.DifficultyMedium, TargetLetterPair: "n-u", TargetLetter: "U", CorrectWord: "UDARA", Distractors: []string{"NDARA", "ADARA", "IDARA"}, Hint: "Kata dengan huruf U, yang kita hirup untuk bernapas"},
	// Hard (abbreviated - add all 18)
	{ID: "h-bd-1", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BERITA", Distractors: []string{"DERITA", "CERITA", "SERITA"}, Hint: "Kata dengan huruf B, informasi atau kabar"},
	{ID: "h-bd-2", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DERITA", Distractors: []string{"BERITA", "CERITA", "SERITA"}, Hint: "Kata dengan huruf D, penderitaan atau kesusahan"},
	{ID: "h-bd-3", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BAKTI", Distractors: []string{"DAKTI", "SAKTI", "FAKTI"}, Hint: "Kata dengan huruf B, pengabdian atau pelayanan"},
	{ID: "h-bd-4", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DALAM", Distractors: []string{"BALAM", "SALAM", "MALAM"}, Hint: "Kata dengan huruf D, lawan dari dangkal atau luar"},
	{ID: "h-bd-5", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "B", CorrectWord: "BUDAYA", Distractors: []string{"DUDAYA", "SUDAYA", "RUDAYA"}, Hint: "Kata dengan huruf B, kebiasaan atau tradisi"},
	{ID: "h-bd-6", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "b-d", TargetLetter: "D", CorrectWord: "DUNIA", Distractors: []string{"BUNIA", "SUNIA", "RUNIA"}, Hint: "Kata dengan huruf D, planet tempat kita tinggal"},
	{ID: "h-mw-1", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MAWAR", Distractors: []string{"WAWAR", "SAWAR", "TAWAR"}, Hint: "Kata dengan huruf M, bunga berduri yang indah"},
	{ID: "h-mw-2", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WAJIB", Distractors: []string{"MAJIB", "SAJIB", "TAJIB"}, Hint: "Kata dengan huruf W, harus dilakukan"},
	{ID: "h-mw-3", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "m-w", TargetLetter: "M", CorrectWord: "MIMPI", Distractors: []string{"WIMPI", "SIMPI", "TIMPI"}, Hint: "Kata dengan huruf M, angan-angan saat tidur"},
	{ID: "h-mw-4", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "m-w", TargetLetter: "W", CorrectWord: "WAJAH", Distractors: []string{"MAJAH", "RAJAH", "SAJAH"}, Hint: "Kata dengan huruf W, muka atau rupa"},
	{ID: "h-pq-1", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PAHAM", Distractors: []string{"QAHAM", "SAHAM", "RAHAM"}, Hint: "Kata dengan huruf P, mengerti atau memahami"},
	{ID: "h-pq-2", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "p-q", TargetLetter: "P", CorrectWord: "PIDATO", Distractors: []string{"QIDATO", "SIDATO", "RIDATO"}, Hint: "Kata dengan huruf P, berbicara di depan umum"},
	{ID: "h-nu-1", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NEGARA", Distractors: []string{"UEGARA", "SEGARA", "MEGARA"}, Hint: "Kata dengan huruf N, Indonesia adalah sebuah..."},
	{ID: "h-nu-2", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "n-u", TargetLetter: "N", CorrectWord: "NAFAS", Distractors: []string{"UAFAS", "RAFAS", "KAFAS"}, Hint: "Kata dengan huruf N, udara yang masuk dan keluar"},
	{ID: "h-nu-3", Difficulty: oldEntity.DifficultyHard, TargetLetterPair: "n-u", TargetLetter: "U", CorrectWord: "UCAPAN", Distractors: []string{"NCAPAN", "ACAPAN", "ICAPAN"}, Hint: "Kata dengan huruf U, kata-kata yang disampaikan"},
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
			Hint:             tpl.Hint,
		}

		if err := db.Create(&template).Error; err != nil {
			return fmt.Errorf("failed to seed template %s: %w", tpl.ID, err)
		}
	}

	fmt.Printf("Successfully seeded %d question bank templates\n", len(QuestionBankData))
	return nil
}
