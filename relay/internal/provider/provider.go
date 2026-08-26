package provider

import (
	"context"
	"github.com/guojunhao/ai-eng-lab/relay/internal/verdict"
)

// Request defines a model request.
type Request struct {
	Prompt string
	Model  string
}

// Response defines a model response.
type Response struct {
	Evidence verdict.Evidence
}

// Provider defines the interface for an AI provider.
type Provider interface {
	Name() string
	Invoke(ctx context.Context, req Request) (Response, error)
}

// MockProvider is a fake provider for testing.
type MockProvider struct {
	ProviderName string
	InvokeFunc   func(ctx context.Context, req Request) (Response, error)
}

// Name returns the provider's name.
func (m *MockProvider) Name() string {
	return m.ProviderName
}

// Invoke calls the mock invoke function.
func (m *MockProvider) Invoke(ctx context.Context, req Request) (Response, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, req)
	}
	return Response{}, nil
}
