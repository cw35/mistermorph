package agent

import "strings"

const (
	PlanStatusPending    = "pending"
	PlanStatusInProgress = "in_progress"
	PlanStatusCompleted  = "completed"
)

func NormalizePlanSteps(p *Plan) {
	if p == nil {
		return
	}

	normalized := make(PlanSteps, 0, len(p.Steps))
	for i := range p.Steps {
		step := strings.TrimSpace(p.Steps[i].Step)
		if step == "" {
			continue
		}
		st := strings.ToLower(strings.TrimSpace(p.Steps[i].Status))
		if st != PlanStatusPending && st != PlanStatusInProgress && st != PlanStatusCompleted {
			st = PlanStatusPending
		}
		normalized = append(normalized, PlanStep{
			Step:   step,
			Status: st,
		})
	}
	p.Steps = normalized
	if len(p.Steps) == 0 {
		return
	}

	// Ensure exactly one in_progress step if there are incomplete steps.
	inProgress := -1
	firstPending := -1
	for i := range p.Steps {
		switch p.Steps[i].Status {
		case PlanStatusInProgress:
			if inProgress == -1 {
				inProgress = i
			} else {
				// multiple in_progress -> demote extras
				p.Steps[i].Status = PlanStatusPending
			}
		case PlanStatusPending:
			if firstPending == -1 {
				firstPending = i
			}
		}
	}
	if inProgress == -1 && firstPending != -1 {
		p.Steps[firstPending].Status = PlanStatusInProgress
	}
}

func AdvancePlanOnSuccess(p *Plan) (completedIndex int, completedStep string, startedIndex int, startedStep string, ok bool) {
	completedIndex = -1
	startedIndex = -1
	if p == nil || len(p.Steps) == 0 {
		return -1, "", -1, "", false
	}

	cur := -1
	next := -1
	for i := range p.Steps {
		if p.Steps[i].Status == PlanStatusInProgress && cur == -1 {
			cur = i
			continue
		}
		if next == -1 && p.Steps[i].Status == PlanStatusPending {
			next = i
		}
	}

	if cur != -1 {
		completedIndex = cur
		completedStep = p.Steps[cur].Step
		p.Steps[cur].Status = PlanStatusCompleted
	}
	if next != -1 {
		startedIndex = next
		startedStep = p.Steps[next].Step
		p.Steps[next].Status = PlanStatusInProgress
	}
	return completedIndex, completedStep, startedIndex, startedStep, cur != -1
}

func CompleteAllPlanSteps(p *Plan) {
	if p == nil {
		return
	}
	for i := range p.Steps {
		p.Steps[i].Status = PlanStatusCompleted
	}
}
