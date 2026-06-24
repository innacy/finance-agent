package categorizer

import (
	"context"
	"regexp"
	"strings"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/ai"
	"github.com/innacy/finance-agent/pkg/brain"
)

type DB interface {
	LookupMerchant(ctx context.Context, userID, normalized string) (*models.MerchantMemory, error)
	GetAllMerchantMemory(ctx context.Context, userID string) ([]models.MerchantMemory, error)
	UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error
	GetCategoryRules(ctx context.Context, userID string) ([]models.CategoryRule, error)
	GetCategories(ctx context.Context, userID string) ([]models.Category, error)
}

type CategorizeInput struct {
	Merchant    string
	Description string
	Channel     string
	Type        string
	Amount      float64
}

type CategorizeResult struct {
	Category    string
	SubCategory string
	Confidence  float64
	Method      string
	NeedsReview bool
}

type Categorizer struct {
	db            DB
	userID        string
	minConfidence float64
	classifier    *brain.Classifier
	aiClient      ai.AIClient
	aiThreshold   float64
}

func New(db DB, userID string, minConfidence float64) *Categorizer {
	return &Categorizer{db: db, userID: userID, minConfidence: minConfidence}
}

func NewWithML(db DB, userID string, minConfidence float64, classifier *brain.Classifier) *Categorizer {
	return &Categorizer{db: db, userID: userID, minConfidence: minConfidence, classifier: classifier}
}

func NewWithAI(db DB, userID string, minConfidence float64, classifier *brain.Classifier, aiClient ai.AIClient, aiThreshold float64) *Categorizer {
	return &Categorizer{
		db: db, userID: userID, minConfidence: minConfidence,
		classifier: classifier, aiClient: aiClient, aiThreshold: aiThreshold,
	}
}

func (c *Categorizer) Categorize(ctx context.Context, input *CategorizeInput) CategorizeResult {
	if result := c.tryPattern(input); result != nil {
		return *result
	}

	if result := c.tryMerchantMemory(ctx, input); result != nil {
		return *result
	}

	if result := c.tryFuzzyMatch(ctx, input); result != nil {
		return *result
	}

	if result := c.tryRules(ctx, input); result != nil {
		return *result
	}

	if result := c.tryML(input); result != nil {
		return *result
	}

	if result := c.tryAI(ctx, input); result != nil {
		return *result
	}

	if result := c.tryKeywords(ctx, input); result != nil {
		return *result
	}

	return CategorizeResult{
		Category:   "Uncategorized",
		Confidence: 0,
		Method:     "none",
	}
}

func (c *Categorizer) Learn(ctx context.Context, merchant, category, source string) error {
	normalized := normalize(merchant)
	return c.db.UpsertMerchantMemory(ctx, &models.MerchantMemory{
		UserID:         c.userID,
		MerchantName:   merchant,
		NormalizedName: normalized,
		Category:       category,
		Confidence:     1.0,
		TimesUsed:      1,
		Source:         source,
	})
}

func (c *Categorizer) tryPattern(input *CategorizeInput) *CategorizeResult {
	if input.Channel == "ATM" {
		return &CategorizeResult{Category: "ATM", Confidence: 1.0, Method: "pattern"}
	}

	if input.Type == "credit" && input.Channel == "NEFT" && input.Amount >= 25000 {
		return &CategorizeResult{Category: "Salary", Confidence: 0.85, Method: "pattern"}
	}

	return nil
}

func (c *Categorizer) tryMerchantMemory(ctx context.Context, input *CategorizeInput) *CategorizeResult {
	if input.Merchant == "" {
		return nil
	}

	normalized := normalize(input.Merchant)
	mem, err := c.db.LookupMerchant(ctx, c.userID, normalized)
	if err != nil || mem == nil {
		return nil
	}

	return &CategorizeResult{
		Category:    mem.Category,
		SubCategory: mem.SubCategory,
		Confidence:  mem.Confidence,
		Method:      "merchant_memory",
	}
}

func (c *Categorizer) tryFuzzyMatch(ctx context.Context, input *CategorizeInput) *CategorizeResult {
	if input.Merchant == "" {
		return nil
	}

	normalized := normalize(input.Merchant)
	allMerchants, err := c.db.GetAllMerchantMemory(ctx, c.userID)
	if err != nil {
		return nil
	}

	var bestMatch *models.MerchantMemory
	bestDistance := 3 // max allowed edits

	for i, mem := range allMerchants {
		dist := levenshtein(normalized, mem.NormalizedName)
		if dist > 0 && dist < bestDistance && dist <= len(normalized)/3 {
			bestDistance = dist
			bestMatch = &allMerchants[i]
		}
	}

	if bestMatch != nil {
		confidence := bestMatch.Confidence * (1.0 - float64(bestDistance)*0.15)
		return &CategorizeResult{
			Category:    bestMatch.Category,
			SubCategory: bestMatch.SubCategory,
			Confidence:  confidence,
			Method:      "fuzzy_match",
		}
	}

	return nil
}

func (c *Categorizer) tryRules(ctx context.Context, input *CategorizeInput) *CategorizeResult {
	rules, err := c.db.GetCategoryRules(ctx, c.userID)
	if err != nil {
		return nil
	}

	for _, rule := range rules {
		var target string
		switch rule.Field {
		case "merchant":
			target = strings.ToLower(input.Merchant)
		case "description":
			target = strings.ToLower(input.Description)
		default:
			target = strings.ToLower(input.Merchant + " " + input.Description)
		}

		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(target) {
			return &CategorizeResult{
				Category:    rule.Category,
				SubCategory: rule.SubCategory,
				Confidence:  0.9,
				Method:      "rule",
			}
		}
	}

	return nil
}

func (c *Categorizer) tryAI(ctx context.Context, input *CategorizeInput) *CategorizeResult {
	if c.aiClient == nil {
		return nil
	}

	cats, _ := c.db.GetCategories(ctx, c.userID)
	catNames := make([]string, 0, len(cats))
	for _, cat := range cats {
		catNames = append(catNames, cat.Name)
	}

	result, err := c.aiClient.CategorizeTransaction(ctx, ai.CategorizeRequest{
		Merchant:    input.Merchant,
		Description: input.Description,
		Amount:      input.Amount,
		Type:        input.Type,
		Channel:     input.Channel,
		Categories:  catNames,
	})
	if err != nil {
		return nil
	}

	if result.Confidence < c.aiThreshold {
		return nil
	}

	return &CategorizeResult{
		Category:   result.Category,
		Confidence: result.Confidence,
		Method:     "ai",
	}
}

func (c *Categorizer) tryML(input *CategorizeInput) *CategorizeResult {
	if c.classifier == nil {
		return nil
	}

	text := strings.Join([]string{input.Merchant, input.Description, input.Channel}, " ")
	category, confidence := c.classifier.Predict(text)

	if confidence < c.minConfidence {
		return nil
	}

	return &CategorizeResult{
		Category:   category,
		Confidence: confidence,
		Method:     "ml",
	}
}

func (c *Categorizer) tryKeywords(ctx context.Context, input *CategorizeInput) *CategorizeResult {
	cats, err := c.db.GetCategories(ctx, c.userID)
	if err != nil {
		return nil
	}

	combined := strings.ToLower(input.Merchant + " " + input.Description)

	for _, cat := range cats {
		for _, kw := range cat.Keywords {
			if strings.Contains(combined, strings.ToLower(kw)) {
				return &CategorizeResult{
					Category:   cat.Name,
					Confidence: 0.7,
					Method:     "keyword",
				}
			}
		}
	}

	return nil
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "")
	return s
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
