package agent

import "testing"

func TestNormalizePlanStepsDropsEmptySteps(t *testing.T) {
	plan := &Plan{
		Steps: PlanSteps{
			{Step: "   ", Status: PlanStatusPending},
			{Step: " collect data ", Status: ""},
			{Step: "", Status: PlanStatusInProgress},
			{Step: "summarize", Status: "unknown"},
		},
	}

	NormalizePlanSteps(plan)

	if len(plan.Steps) != 2 {
		t.Fatalf("len(plan.Steps) = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].Step != "collect data" {
		t.Fatalf("steps[0].step = %q, want %q", plan.Steps[0].Step, "collect data")
	}
	if plan.Steps[0].Status != PlanStatusInProgress {
		t.Fatalf("steps[0].status = %q, want %q", plan.Steps[0].Status, PlanStatusInProgress)
	}
	if plan.Steps[1].Step != "summarize" {
		t.Fatalf("steps[1].step = %q, want %q", plan.Steps[1].Step, "summarize")
	}
	if plan.Steps[1].Status != PlanStatusPending {
		t.Fatalf("steps[1].status = %q, want %q", plan.Steps[1].Status, PlanStatusPending)
	}
}

func TestAdvancePlanOnSuccessNoInProgressStepReturnsFalse(t *testing.T) {
	plan := &Plan{
		Steps: PlanSteps{
			{Step: "done", Status: PlanStatusCompleted},
		},
	}

	completedIndex, completedStep, startedIndex, startedStep, ok := AdvancePlanOnSuccess(plan)
	if ok {
		t.Fatal("ok = true, want false")
	}
	if completedIndex != -1 || completedStep != "" {
		t.Fatalf("completed = (%d, %q), want (-1, \"\")", completedIndex, completedStep)
	}
	if startedIndex != -1 || startedStep != "" {
		t.Fatalf("started = (%d, %q), want (-1, \"\")", startedIndex, startedStep)
	}
}

func TestAdvancePlanOnSuccessAdvancesNormalizedPlan(t *testing.T) {
	plan := &Plan{
		Steps: PlanSteps{
			{Step: "collect data", Status: PlanStatusInProgress},
			{Step: "summarize", Status: PlanStatusPending},
		},
	}

	completedIndex, completedStep, startedIndex, startedStep, ok := AdvancePlanOnSuccess(plan)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if completedIndex != 0 || completedStep != "collect data" {
		t.Fatalf("completed = (%d, %q), want (0, %q)", completedIndex, completedStep, "collect data")
	}
	if startedIndex != 1 || startedStep != "summarize" {
		t.Fatalf("started = (%d, %q), want (1, %q)", startedIndex, startedStep, "summarize")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("len(plan.Steps) = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].Status != PlanStatusCompleted {
		t.Fatalf("steps[0].status = %q, want %q", plan.Steps[0].Status, PlanStatusCompleted)
	}
	if plan.Steps[1].Status != PlanStatusInProgress {
		t.Fatalf("steps[1].status = %q, want %q", plan.Steps[1].Status, PlanStatusInProgress)
	}
}
