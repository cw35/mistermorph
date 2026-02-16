package grouptrigger

import (
	"context"
	"strings"
	"time"
)

type Decision struct {
	Reason            string
	UsedAddressingLLM bool

	AddressingLLMAttempted  bool
	AddressingLLMOK         bool
	AddressingLLMAddressed  bool
	AddressingLLMConfidence float64
	AddressingLLMInterject  float64
	AddressingImpulse       float64
}

type AddressingDecision struct {
	Addressed  bool
	Confidence float64
	Interject  float64
	Impulse    float64
	Reason     string
}

type AddressingFunc func(ctx context.Context) (AddressingDecision, bool, error)

type DecideOptions struct {
	Mode                     string
	DefaultMode              string
	ConfidenceThreshold      float64
	InterjectThreshold       float64
	DefaultConfidence        float64
	DefaultInterject         float64
	ExplicitReason           string
	ExplicitMatched          bool
	AddressingFallbackReason string
	AddressingTimeout        time.Duration
	Addressing               AddressingFunc
}

func Decide(ctx context.Context, opts DecideOptions) (Decision, bool, error) {
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	defaultMode := strings.ToLower(strings.TrimSpace(opts.DefaultMode))
	if defaultMode == "" {
		defaultMode = "smart"
	}
	if mode == "" {
		mode = defaultMode
	}

	confidenceThreshold := opts.ConfidenceThreshold
	if confidenceThreshold <= 0 {
		confidenceThreshold = opts.DefaultConfidence
	}
	if confidenceThreshold <= 0 {
		confidenceThreshold = 0.6
	}
	confidenceThreshold = clamp01(confidenceThreshold)

	interjectThreshold := opts.InterjectThreshold
	if interjectThreshold <= 0 {
		interjectThreshold = opts.DefaultInterject
	}
	if interjectThreshold <= 0 {
		interjectThreshold = 0.6
	}
	interjectThreshold = clamp01(interjectThreshold)

	if opts.ExplicitMatched {
		return Decision{
			Reason:            strings.TrimSpace(opts.ExplicitReason),
			AddressingImpulse: 1,
		}, true, nil
	}

	runAddressingLLM := func(requireAddressed bool) (Decision, bool, error) {
		dec := Decision{
			AddressingLLMAttempted: true,
			Reason:                 strings.TrimSpace(opts.AddressingFallbackReason),
		}
		if opts.Addressing == nil {
			return dec, false, nil
		}
		addrCtx := ctx
		if addrCtx == nil {
			addrCtx = context.Background()
		}
		cancel := func() {}
		if opts.AddressingTimeout > 0 {
			addrCtx, cancel = context.WithTimeout(addrCtx, opts.AddressingTimeout)
		}
		llmDec, llmOK, llmErr := opts.Addressing(addrCtx)
		cancel()
		if llmErr != nil {
			return dec, false, llmErr
		}
		llmDec.Confidence = clamp01(llmDec.Confidence)
		llmDec.Interject = clamp01(llmDec.Interject)
		llmDec.Impulse = clamp01(llmDec.Impulse)

		dec.AddressingLLMOK = llmOK
		dec.AddressingLLMAddressed = llmDec.Addressed
		dec.AddressingLLMConfidence = llmDec.Confidence
		dec.AddressingLLMInterject = llmDec.Interject
		dec.AddressingImpulse = llmDec.Impulse
		if strings.TrimSpace(llmDec.Reason) != "" {
			dec.Reason = strings.TrimSpace(llmDec.Reason)
		}

		addressedOK := true
		if requireAddressed {
			addressedOK = llmDec.Addressed
		}
		if llmOK && addressedOK && llmDec.Confidence >= confidenceThreshold && llmDec.Interject > interjectThreshold {
			dec.UsedAddressingLLM = true
			return dec, true, nil
		}
		return dec, false, nil
	}

	switch mode {
	case "talkative":
		return runAddressingLLM(false)
	case "smart":
		return runAddressingLLM(true)
	default:
		return Decision{}, false, nil
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
