package router

import "strings"

// ShouldRouteToSmallModel decides if a query is low-value/repetitive and should be routed to a smaller model.
func ShouldRouteToSmallModel(prompt string) bool {
	if len(prompt) < 20 {
		return true
	}
	
	lower := strings.ToLower(prompt)
	keywords := []string{"hello", "how are you", "what is", "hi there", "good morning"}
	
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	
	return false
}
