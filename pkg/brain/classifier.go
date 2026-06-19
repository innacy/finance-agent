package brain

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strings"
)

type document struct {
	Tokens   []string
	Category string
	Weight   float64
}

type ClassifierStats struct {
	TotalDocuments int
	Categories     int
	VocabSize      int
}

type Classifier struct {
	docs       []document
	catCounts  map[string]float64
	wordCounts map[string]map[string]float64
	vocab      map[string]int
	totalDocs  float64
	docFreq    map[string]int
}

func NewClassifier() *Classifier {
	return &Classifier{
		catCounts:  make(map[string]float64),
		wordCounts: make(map[string]map[string]float64),
		vocab:      make(map[string]int),
		docFreq:    make(map[string]int),
	}
}

func (c *Classifier) Train(text, category string, weight float64) {
	tokens := tokenize(text)
	c.docs = append(c.docs, document{Tokens: tokens, Category: category, Weight: weight})

	c.catCounts[category] += weight
	c.totalDocs += weight

	if c.wordCounts[category] == nil {
		c.wordCounts[category] = make(map[string]float64)
	}

	seen := make(map[string]bool)
	for _, tok := range tokens {
		c.wordCounts[category][tok] += weight
		c.vocab[tok]++
		if !seen[tok] {
			c.docFreq[tok]++
			seen[tok] = true
		}
	}
}

func (c *Classifier) Predict(text string) (string, float64) {
	if c.totalDocs == 0 || len(c.catCounts) == 0 {
		return "Uncategorized", 0
	}

	tokens := tokenize(text)
	tfidfScores := c.tfidf(text)

	scores := make(map[string]float64)
	for cat, catCount := range c.catCounts {
		logProb := math.Log(catCount / c.totalDocs)

		wordTotal := 0.0
		for _, count := range c.wordCounts[cat] {
			wordTotal += count
		}

		vocabSize := float64(len(c.vocab))

		for _, tok := range tokens {
			wordCount := c.wordCounts[cat][tok]
			prob := (wordCount + 1.0) / (wordTotal + vocabSize)
			tfidfWeight := tfidfScores[tok]
			if tfidfWeight == 0 {
				tfidfWeight = 1.0
			}
			logProb += math.Log(prob) * tfidfWeight
		}

		scores[cat] = logProb
	}

	bestCat := "Uncategorized"
	bestScore := math.Inf(-1)
	for cat, score := range scores {
		if score > bestScore {
			bestScore = score
			bestCat = cat
		}
	}

	confidence := c.computeConfidence(scores, bestCat)
	return bestCat, confidence
}

func (c *Classifier) tfidf(text string) map[string]float64 {
	tokens := tokenize(text)
	tf := make(map[string]float64)
	for _, tok := range tokens {
		tf[tok]++
	}

	totalTokens := float64(len(tokens))
	totalDocsF := float64(len(c.docs))
	if totalDocsF == 0 {
		totalDocsF = 1
	}

	scores := make(map[string]float64)
	for tok, count := range tf {
		termFreq := count / totalTokens
		df := float64(c.docFreq[tok])
		if df == 0 {
			df = 1
		}
		idf := math.Log(totalDocsF/df) + 1
		scores[tok] = termFreq * idf
	}
	return scores
}

func (c *Classifier) computeConfidence(scores map[string]float64, bestCat string) float64 {
	if len(scores) <= 1 {
		if c.totalDocs < 5 {
			return 0.3
		}
		return 0.7
	}

	bestScore := scores[bestCat]
	secondBest := math.Inf(-1)
	for cat, score := range scores {
		if cat != bestCat && score > secondBest {
			secondBest = score
		}
	}

	diff := bestScore - secondBest
	confidence := 1.0 / (1.0 + math.Exp(-diff))

	if c.totalDocs < 10 {
		confidence *= 0.8
	} else if c.totalDocs < 30 {
		confidence *= 0.9
	}

	return math.Min(confidence, 0.99)
}

func (c *Classifier) Stats() ClassifierStats {
	return ClassifierStats{
		TotalDocuments: len(c.docs),
		Categories:     len(c.catCounts),
		VocabSize:      len(c.vocab),
	}
}

type persistedModel struct {
	Docs       []document                    `json:"docs"`
	CatCounts  map[string]float64            `json:"cat_counts"`
	WordCounts map[string]map[string]float64 `json:"word_counts"`
	Vocab      map[string]int                `json:"vocab"`
	TotalDocs  float64                       `json:"total_docs"`
	DocFreq    map[string]int                `json:"doc_freq"`
}

func (c *Classifier) Save(path string) error {
	model := persistedModel{
		Docs:       c.docs,
		CatCounts:  c.catCounts,
		WordCounts: c.wordCounts,
		Vocab:      c.vocab,
		TotalDocs:  c.totalDocs,
		DocFreq:    c.docFreq,
	}

	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Classifier) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var model persistedModel
	if err := json.Unmarshal(data, &model); err != nil {
		return err
	}

	c.docs = model.Docs
	c.catCounts = model.CatCounts
	c.wordCounts = model.WordCounts
	c.vocab = model.Vocab
	c.totalDocs = model.TotalDocs
	c.docFreq = model.DocFreq
	return nil
}

var reTokenize = regexp.MustCompile(`[a-z0-9]+`)

func tokenize(text string) []string {
	text = strings.ToLower(text)
	matches := reTokenize.FindAllString(text, -1)

	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			tokens = append(tokens, m)
		}
	}
	return tokens
}
