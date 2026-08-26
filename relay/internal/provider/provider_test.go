package provider

import (
	"context"
	"errors"
	"testing"
	"github.com/guojunhao/ai-eng-lab/relay/internal/verdict"
)

func TestMockProvider(t *testing.T) {
	mock := &MockProvider{
		ProviderName: "mock-provider",
		InvokeFunc: func(ctx context.Context, req Request) (Response, error) {
			if req.Model == "fail-model" {
				return Response{}, errors.New("simulated error")
			}
			return Response{
				Evidence: verdict.Evidence{
					ExitCode: 0,
					Stdout:   "success",
				},
			}, nil
		},
	}

	if mock.Name() != "mock-provider" {
		t.Errorf("expected mock-provider, got %v", mock.Name())
	}

	ctx := context.Background()
	_, err := mock.Invoke(ctx, Request{Model: "fail-model"})
	if err == nil {
		t.Errorf("expected error for fail-model")
	}

	resp, err := mock.Invoke(ctx, Request{Model: "good-model"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.Evidence.Stdout != "success" {
		t.Errorf("expected success, got %v", resp.Evidence.Stdout)
	}
}
