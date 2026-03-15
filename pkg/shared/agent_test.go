package shared

import (
	"testing"
)

func TestLLMClientConfigProvider(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		want    LLMProvider
		wantErr error
	}{
		{
			name:    "anthropic provider",
			model:   "anthropic/claude-3-5-sonnet-20241022",
			want:    LLMProviderAnthropic,
			wantErr: nil,
		},
		{
			name:    "openai provider",
			model:   "openai/gpt-4o",
			want:    LLMProviderOpenAI,
			wantErr: nil,
		},
		{
			name:    "gemini provider",
			model:   "gemini/gemini-2.0-flash-exp",
			want:    LLMProviderGemini,
			wantErr: nil,
		},
		{
			name:    "no slash - invalid format",
			model:   "claude-3-5-sonnet-20241022",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "empty string - invalid format",
			model:   "",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "only slash - invalid format",
			model:   "/",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "unsupported provider",
			model:   "unknown/some-model",
			want:    "",
			wantErr: ErrLLMProviderNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnf := &LLMClientConfig{Model: tt.model}
			got, err := cnf.Provider()

			if err != tt.wantErr {
				t.Errorf("Provider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Provider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLLMClientConfigModelName(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		want    string
		wantErr error
	}{
		{
			name:    "anthropic model",
			model:   "anthropic/claude-3-5-sonnet-20241022",
			want:    "claude-3-5-sonnet-20241022",
			wantErr: nil,
		},
		{
			name:    "openai model",
			model:   "openai/gpt-4o",
			want:    "gpt-4o",
			wantErr: nil,
		},
		{
			name:    "gemini model",
			model:   "gemini/gemini-2.0-flash-exp",
			want:    "gemini-2.0-flash-exp",
			wantErr: nil,
		},
		{
			name:    "no slash - invalid format",
			model:   "claude-3-5-sonnet-20241022",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "empty string - invalid format",
			model:   "",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "trailing slash",
			model:   "anthropic/",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "leading slash - invalid format",
			model:   "/model",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnf := &LLMClientConfig{Model: tt.model}
			got, err := cnf.ModelName()

			if err != tt.wantErr {
				t.Errorf("ModelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ModelName() = %v, want %v", got, tt.want)
			}
		})
	}
}
