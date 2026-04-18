package policy

import "testing"

func TestVADPolicyDecide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decision VoiceDecision
		want     AudioAction
	}{
		{
			name: "speech flag passes audio",
			decision: VoiceDecision{
				IsSpeech: true,
				State:    VoiceStateSilence,
			},
			want: AudioActionPassThrough,
		},
		{
			name: "speech state passes audio",
			decision: VoiceDecision{
				State: VoiceStateSpeech,
			},
			want: AudioActionPassThrough,
		},
		{
			name: "maybe silence hangover passes audio",
			decision: VoiceDecision{
				State: VoiceStateMaybeSilence,
			},
			want: AudioActionPassThrough,
		},
		{
			name: "silence silences audio",
			decision: VoiceDecision{
				State: VoiceStateSilence,
			},
			want: AudioActionSilence,
		},
	}

	policy := VADPolicy{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := policy.Decide(tt.decision); got != tt.want {
				t.Fatalf("Decide() = %s, want %s", got, tt.want)
			}
		})
	}
}
