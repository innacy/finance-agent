package brain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrainAndPredict(t *testing.T) {
	c := NewClassifier()

	c.Train("swiggy food delivery order", "Food & Dining", 1.0)
	c.Train("zomato restaurant dinner", "Food & Dining", 1.0)
	c.Train("uber ride trip", "Transport", 1.0)
	c.Train("ola cab booking", "Transport", 1.0)
	c.Train("amazon shopping purchase", "Shopping", 1.0)
	c.Train("flipkart online order", "Shopping", 1.0)

	category, confidence := c.Predict("swiggy delivery food")
	if category != "Food & Dining" {
		t.Errorf("expected Food & Dining, got %q", category)
	}
	if confidence < 0.5 {
		t.Errorf("expected confidence > 0.5, got %f", confidence)
	}
}

func TestPredictWithWeightedTraining(t *testing.T) {
	c := NewClassifier()

	c.Train("random vendor xyz payment", "Shopping", 1.0)
	c.Train("random vendor xyz payment", "Food & Dining", 10.0)

	category, _ := c.Predict("random vendor xyz payment")
	if category != "Food & Dining" {
		t.Errorf("weighted correction (10x) should override, got %q", category)
	}
}

func TestPredictMinimumSamples(t *testing.T) {
	c := NewClassifier()

	c.Train("one sample only", "Food", 1.0)

	_, confidence := c.Predict("something random")
	if confidence > 0.5 {
		t.Errorf("with minimal training, confidence should be low, got %f", confidence)
	}
}

func TestPersistAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brain.model")

	c := NewClassifier()
	c.Train("swiggy food delivery", "Food & Dining", 1.0)
	c.Train("uber ride trip", "Transport", 1.0)
	c.Train("amazon shopping", "Shopping", 1.0)

	err := c.Save(path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("model file should exist after save")
	}

	loaded := NewClassifier()
	err = loaded.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	category, conf := loaded.Predict("swiggy food order")
	if category != "Food & Dining" {
		t.Errorf("loaded model should predict Food & Dining, got %q", category)
	}
	if conf < 0.5 {
		t.Errorf("loaded model confidence should be > 0.5, got %f", conf)
	}
}

func TestTokenize(t *testing.T) {
	tokens := tokenize("UPI-SWIGGY food delivery @123")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}

	for _, tok := range tokens {
		if tok == "" {
			t.Error("empty token found")
		}
		for _, ch := range tok {
			if ch >= 'A' && ch <= 'Z' {
				t.Errorf("token %q should be lowercase", tok)
			}
		}
	}
}

func TestTFIDF(t *testing.T) {
	c := NewClassifier()
	c.Train("swiggy food delivery", "Food", 1.0)
	c.Train("uber ride trip", "Transport", 1.0)
	c.Train("swiggy dinner order", "Food", 1.0)

	scores := c.tfidf("swiggy food")
	if len(scores) == 0 {
		t.Fatal("expected TF-IDF scores")
	}
	if scores["swiggy"] <= 0 {
		t.Error("swiggy should have positive TF-IDF score")
	}
}

func TestClassifierStats(t *testing.T) {
	c := NewClassifier()
	c.Train("food one", "Food", 1.0)
	c.Train("food two", "Food", 1.0)
	c.Train("transport one", "Transport", 1.0)

	stats := c.Stats()
	if stats.TotalDocuments != 3 {
		t.Errorf("expected 3 docs, got %d", stats.TotalDocuments)
	}
	if stats.Categories != 2 {
		t.Errorf("expected 2 categories, got %d", stats.Categories)
	}
	if stats.VocabSize == 0 {
		t.Error("expected non-zero vocab size")
	}
}
