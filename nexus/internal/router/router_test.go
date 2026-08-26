package router

import "testing"

func TestShouldRouteToSmallModel(t *testing.T) {
	if !ShouldRouteToSmallModel("hi") {
		t.Error("expected true for short prompt")
	}
}

func TestRouter_ShortPrompt(t *testing.T) {
	if !ShouldRouteToSmallModel("1234567890123456789") { // 19 chars
		t.Error("19 chars should route to small")
	}
	if ShouldRouteToSmallModel("12345678901234567890") { // 20 chars
		t.Error("20 chars should not route to small if no keyword")
	}
}

func TestRouter_KeywordHello(t *testing.T) {
	if !ShouldRouteToSmallModel("Hello, could you help me with something complex?") {
		t.Error("Should route due to 'hello'")
	}
}

func TestRouter_KeywordHowAreYou(t *testing.T) {
	if !ShouldRouteToSmallModel("Well, how are you today, friend?") {
		t.Error("Should route due to 'how are you'")
	}
}

func TestRouter_KeywordWhatIs(t *testing.T) {
	if !ShouldRouteToSmallModel("I was wondering what is the meaning of life?") {
		t.Error("Should route due to 'what is'")
	}
}

func TestRouter_NoKeyword(t *testing.T) {
	if ShouldRouteToSmallModel("Please explain quantum field theory in detail.") {
		t.Error("Should not route to small")
	}
}

func TestRouter_EmptyPrompt(t *testing.T) {
	if !ShouldRouteToSmallModel("") {
		t.Error("Empty prompt should route to small")
	}
}

func TestRouter_Exactly20Chars(t *testing.T) {
	if ShouldRouteToSmallModel("12345678901234567890") {
		t.Error("20 chars exactly should not route to small if no keyword")
	}
}

func TestRouter_KeywordUppercase(t *testing.T) {
	if !ShouldRouteToSmallModel("HELLO, COULD YOU HELP ME?") {
		t.Error("Uppercase keyword should route")
	}
}
