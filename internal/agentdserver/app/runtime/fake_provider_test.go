package runtime

import (
	"context"
	"sync"
)

type fakeProvider struct {
	name     string
	response ProviderResponse
	err      error

	mu       sync.Mutex
	requests []ProviderRequest
}

func newFakeProvider(response ProviderResponse, err error) *fakeProvider {
	return &fakeProvider{name: "fake", response: response, err: err}
}

func (p *fakeProvider) Name() string {
	if p.name == "" {
		return "fake"
	}

	return p.name
}

func (p *fakeProvider) Execute(
	_ context.Context,
	request ProviderRequest,
) (ProviderResponse, error) {
	p.mu.Lock()
	p.requests = append(p.requests, request)
	p.mu.Unlock()

	if p.err != nil {
		return ProviderResponse{}, p.err
	}

	return p.response, nil
}

func (p *fakeProvider) Requests() []ProviderRequest {
	p.mu.Lock()
	defer p.mu.Unlock()

	copied := make([]ProviderRequest, len(p.requests))
	copy(copied, p.requests)

	return copied
}
