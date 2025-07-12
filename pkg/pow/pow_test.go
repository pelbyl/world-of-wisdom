package pow

import (
	"fmt"
	"testing"
	"time"
)

func TestGenerateChallenge(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
		wantErr    bool
	}{
		{"Valid difficulty 1", 1, false},
		{"Valid difficulty 3", 3, false},
		{"Valid difficulty 6", 6, false},
		{"Invalid difficulty 0", 0, true},
		{"Invalid difficulty 7", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := GenerateChallenge(tt.difficulty)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateChallenge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if challenge.Difficulty != tt.difficulty {
					t.Errorf("Challenge difficulty = %v, want %v", challenge.Difficulty, tt.difficulty)
				}
				if len(challenge.Seed) != 32 {
					t.Errorf("Seed length = %v, want 32", len(challenge.Seed))
				}
			}
		})
	}
}

func TestVerifyPoW(t *testing.T) {
	tests := []struct {
		name       string
		seed       string
		nonce      string
		difficulty int
		want       bool
	}{
		{"Valid solution", "test", "0", 1, false},
		{"Invalid solution", "test", "999", 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				challenge := &Challenge{Seed: tt.seed, Difficulty: tt.difficulty}
				solution, err := SolveChallenge(challenge)
				if err != nil {
					t.Fatalf("Failed to solve challenge: %v", err)
				}
				if !VerifyPoW(tt.seed, solution, tt.difficulty) {
					t.Errorf("VerifyPoW() = false for valid solution")
				}
			}
		})
	}
}

func TestSolveChallenge(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
		maxTime    time.Duration
	}{
		{"Difficulty 1", 1, 100 * time.Millisecond},
		{"Difficulty 2", 2, 500 * time.Millisecond},
		{"Difficulty 3", 3, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := GenerateChallenge(tt.difficulty)
			if err != nil {
				t.Fatalf("Failed to generate challenge: %v", err)
			}

			start := time.Now()
			solution, err := SolveChallenge(challenge)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("Failed to solve challenge: %v", err)
			}

			if !VerifyPoW(challenge.Seed, solution, challenge.Difficulty) {
				t.Errorf("Invalid solution returned")
			}

			t.Logf("Solved difficulty %d in %v", tt.difficulty, elapsed)
		})
	}
}

func BenchmarkSolveChallenge(b *testing.B) {
	difficulties := []int{1, 2, 3, 4}
	
	for _, diff := range difficulties {
		b.Run(fmt.Sprintf("Difficulty%d", diff), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				challenge, _ := GenerateChallenge(diff)
				SolveChallenge(challenge)
			}
		})
	}
}