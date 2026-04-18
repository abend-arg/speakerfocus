package policy

type VoiceDecision struct {
	SpeechProbability float32
	IsSpeech          bool
	State             VoiceState
}

type VoiceState string

const (
	VoiceStateSilence      VoiceState = "silence"
	VoiceStateSpeech       VoiceState = "speech"
	VoiceStateMaybeSilence VoiceState = "maybe_silence"
)

type AudioAction string

const (
	AudioActionPassThrough AudioAction = "pass_through"
	AudioActionSilence     AudioAction = "silence"
)

type VADPolicy struct{}

func (VADPolicy) Decide(decision VoiceDecision) AudioAction {
	if decision.IsSpeech || decision.State == VoiceStateSpeech || decision.State == VoiceStateMaybeSilence {
		return AudioActionPassThrough
	}
	return AudioActionSilence
}
