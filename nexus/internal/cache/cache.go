package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	Response  string
	Timestamp time.Time
}

type SemanticEntry struct {
	Vector    []float64
	Response  string
	Timestamp time.Time
	ID        int
}

type Cache struct {
	mu             sync.RWMutex
	PrefixMap      map[string]CacheEntry
	SemanticEntries []SemanticEntry
	InvertedIndex  [64][]int
	Window         time.Duration
}

func NewCache(window time.Duration) *Cache {
	var inv [64][]int
	for i := 0; i < 64; i++ {
		inv[i] = make([]int, 0)
	}
	return &Cache{
		PrefixMap:       make(map[string]CacheEntry),
		SemanticEntries: make([]SemanticEntry, 0),
		InvertedIndex:   inv,
		Window:          window,
	}
}

// embed creates a deterministic 64-dimensional vector based on word hashes.
func embed(text string) []float64 {
	vec := make([]float64, 64)
	words := strings.Fields(strings.ToLower(text))
	for _, w := range words {
		var h uint32
		for i := 0; i < len(w); i++ {
			h = h*31 + uint32(w[i])
		}
		vec[h%64]++
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

func getPrefixHash(prompt string) string {
	prefix := prompt
	if len(prompt) > 50 {
		prefix = prompt[:50]
	}
	hash := sha256.Sum256([]byte(prefix))
	return hex.EncodeToString(hash[:])
}

func (c *Cache) GetPrefix(prompt string, now time.Time) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := getPrefixHash(prompt)
	if entry, ok := c.PrefixMap[key]; ok {
		if now.Sub(entry.Timestamp) <= c.Window {
			return entry.Response, true
		}
	}
	return "", false
}

func (c *Cache) SetPrefix(prompt string, response string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := getPrefixHash(prompt)
	c.PrefixMap[key] = CacheEntry{Response: response, Timestamp: now}
}

func (c *Cache) FindBestMatch(prompt string, now time.Time) (string, float64) {
	vec := embed(prompt)

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find candidates that share at least one non-zero dimension
	visited := make([]bool, len(c.SemanticEntries))
	candidates := make([]int, 0, len(vec)) // Pre-allocate with some small capacity
	
	for i, val := range vec {
		if val > 0 {
			for _, id := range c.InvertedIndex[i] {
				if !visited[id] {
					visited[id] = true
					candidates = append(candidates, id)
				}
			}
		}
	}

	var bestResp string
	var bestScore float64 = -1

	for _, id := range candidates {
		entry := c.SemanticEntries[id]
		if now.Sub(entry.Timestamp) > c.Window {
			continue
		}

		var score float64
		for i := range vec {
			if vec[i] > 0 && entry.Vector[i] > 0 {
				score += vec[i] * entry.Vector[i]
			}
		}

		if score > bestScore {
			bestScore = score
			bestResp = entry.Response
			if bestScore >= 0.9999 {
				break
			}
		}
	}

	return bestResp, bestScore
}

func (c *Cache) GetSemantic(prompt string, threshold float64, now time.Time) (string, bool) {
	bestResp, bestScore := c.FindBestMatch(prompt, now)
	if bestScore >= threshold {
		return bestResp, true
	}
	return "", false
}

func (c *Cache) SetSemantic(prompt string, response string, now time.Time) {
	vec := embed(prompt)

	c.mu.Lock()
	defer c.mu.Unlock()

	id := len(c.SemanticEntries)
	c.SemanticEntries = append(c.SemanticEntries, SemanticEntry{
		Vector:    vec,
		Response:  response,
		Timestamp: now,
		ID:        id,
	})
	
	for i, val := range vec {
		if val > 0 {
			c.InvertedIndex[i] = append(c.InvertedIndex[i], id)
		}
	}
}
