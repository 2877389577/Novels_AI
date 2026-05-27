package aiprovider

import (
	"context"
	"testing"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
)

func TestCreatePersistsTrimmedDefaultModel(t *testing.T) {
	repo := &fakeAIProviderRepo{}
	uc := NewAIProviderUseCase(repo, fakeCipher{})

	_, err := uc.Create(context.Background(), dto.CreateAIProviderRequest{
		Name:         "测试提供商",
		ProviderType: "openai",
		BaseURL:      "https://example.com",
		APIKey:       "test-key",
		DefaultModel: "  doubao-test  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdProvider.DefaultModel != "doubao-test" {
		t.Fatalf("unexpected default model: %q", repo.createdProvider.DefaultModel)
	}
}

func TestUpdatePersistsTrimmedDefaultModel(t *testing.T) {
	repo := &fakeAIProviderRepo{}
	uc := NewAIProviderUseCase(repo, fakeCipher{})

	_, err := uc.Update(context.Background(), dto.UpdateAIProviderRequest{
		ID:           7,
		Name:         "测试提供商",
		ProviderType: "openai",
		BaseURL:      "https://example.com",
		APIKey:       "test-key",
		DefaultModel: "  doubao-update  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedProvider.DefaultModel != "doubao-update" {
		t.Fatalf("unexpected default model: %q", repo.updatedProvider.DefaultModel)
	}
}

type fakeAIProviderRepo struct {
	enabledProvider *data.AIProvider
	createdProvider *data.AIProvider
	updatedProvider *data.AIProvider
}

func (repo *fakeAIProviderRepo) Create(ctx context.Context, provider *data.AIProvider) (*data.AIProvider, error) {
	repo.createdProvider = provider
	return provider, nil
}

func (repo *fakeAIProviderRepo) List(ctx context.Context, offset, limit int) ([]*data.AIProvider, int64, error) {
	return nil, 0, nil
}

func (repo *fakeAIProviderRepo) FindByID(ctx context.Context, id int64) (*data.AIProvider, error) {
	return &data.AIProvider{ID: id, APIKeyEncrypted: "cipher"}, nil
}

func (repo *fakeAIProviderRepo) FindEnabled(ctx context.Context) (*data.AIProvider, error) {
	return repo.enabledProvider, nil
}

func (repo *fakeAIProviderRepo) Update(ctx context.Context, provider *data.AIProvider) (*data.AIProvider, error) {
	repo.updatedProvider = provider
	return provider, nil
}

func (repo *fakeAIProviderRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func (repo *fakeAIProviderRepo) Enable(ctx context.Context, id int64) error {
	return nil
}

type fakeCipher struct{}

func (fakeCipher) Encrypt(plainText string) (string, error) {
	return "cipher:" + plainText, nil
}

func (fakeCipher) Decrypt(cipherText string) (string, error) {
	return "plain", nil
}
