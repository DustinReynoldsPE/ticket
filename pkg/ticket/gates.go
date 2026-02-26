package ticket

import (
	"fmt"
	"strings"
)

// GateCheck describes a precondition for a stage transition.
type GateCheck struct {
	From        Stage
	To          Stage
	Description string
	Check       func(t *Ticket) error
}

// Gates returns the gate checks for advancing from one stage to the next
// for the given ticket type and risk level.
func Gates(ticketType TicketType, risk RiskLevel) []GateCheck {
	gates := baseGates(ticketType)
	return applyRiskScaling(gates, risk)
}

// CheckGates runs all gate checks for advancing ticket t from its current
// stage to the given target stage. Returns all failures.
func CheckGates(t *Ticket, to Stage) []error {
	risk := t.Risk
	if risk == "" {
		risk = RiskNormal
	}
	gates := Gates(t.Type, risk)

	var errs []error
	for _, g := range gates {
		if g.From == t.Stage && g.To == to {
			if err := g.Check(t); err != nil {
				errs = append(errs, fmt.Errorf("gate %q: %w", g.Description, err))
			}
		}
	}
	return errs
}

func baseGates(ticketType TicketType) []GateCheck {
	var gates []GateCheck

	// triage → spec (feature, epic)
	if HasStage(ticketType, StageSpec) {
		gates = append(gates, GateCheck{
			From:        StageTriage,
			To:          StageSpec,
			Description: "description exists",
			Check:       checkDescriptionExists,
		})
	}

	// triage → implement (bug, chore, task)
	if !HasStage(ticketType, StageSpec) && HasStage(ticketType, StageImplement) {
		gates = append(gates, GateCheck{
			From:        StageTriage,
			To:          StageImplement,
			Description: "description exists",
			Check:       checkDescriptionExists,
		})
	}

	// spec → design (feature, epic)
	if HasStage(ticketType, StageSpec) && HasStage(ticketType, StageDesign) {
		gates = append(gates, GateCheck{
			From:        StageSpec,
			To:          StageDesign,
			Description: "acceptance criteria exist",
			Check:       checkACExists,
		},
			GateCheck{
				From:        StageSpec,
				To:          StageDesign,
				Description: "spec review approved",
				Check:       checkReviewApproved,
			})
	}

	// design → implement (feature)
	if HasStage(ticketType, StageDesign) && HasStage(ticketType, StageImplement) {
		gates = append(gates, GateCheck{
			From:        StageDesign,
			To:          StageImplement,
			Description: "design section exists",
			Check:       checkDesignExists,
		},
			GateCheck{
				From:        StageDesign,
				To:          StageImplement,
				Description: "design review approved",
				Check:       checkReviewApproved,
			})
	}

	// design → done (epic)
	if HasStage(ticketType, StageDesign) && !HasStage(ticketType, StageImplement) {
		gates = append(gates, GateCheck{
			From:        StageDesign,
			To:          StageDone,
			Description: "design exists",
			Check:       checkDesignExists,
		},
			GateCheck{
				From:        StageDesign,
				To:          StageDone,
				Description: "design review approved",
				Check:       checkReviewApproved,
			})
	}

	// implement → test (feature, bug, task — mandatory reviews)
	if HasStage(ticketType, StageImplement) && HasStage(ticketType, StageTest) {
		gates = append(gates, GateCheck{
			From:        StageImplement,
			To:          StageTest,
			Description: "code review approved",
			Check:       checkCodeReviewApproved,
		},
			GateCheck{
				From:        StageImplement,
				To:          StageTest,
				Description: "impl review approved",
				Check:       checkImplReviewApproved,
			})
	}

	// implement → done (chore — advisory review surfaced)
	if HasStage(ticketType, StageImplement) && !HasStage(ticketType, StageTest) {
		gates = append(gates, GateCheck{
			From:        StageImplement,
			To:          StageDone,
			Description: "advisory review surfaced",
			Check:       checkAdvisoryReviewSurfaced,
		})
	}

	// test → verify (feature, bug, task)
	if HasStage(ticketType, StageTest) && HasStage(ticketType, StageVerify) {
		gates = append(gates, GateCheck{
			From:        StageTest,
			To:          StageVerify,
			Description: "test results recorded",
			Check:       checkTestResultsRecorded,
		})
	}

	// verify → done (feature, bug, task)
	if HasStage(ticketType, StageVerify) {
		gates = append(gates, GateCheck{
			From:        StageVerify,
			To:          StageDone,
			Description: "verification approved",
			Check:       checkReviewApproved,
		})
	}

	return gates
}

func applyRiskScaling(gates []GateCheck, risk RiskLevel) []GateCheck {
	switch risk {
	case RiskLow:
		// Low risk: make all gates advisory (no-op checks).
		advisory := make([]GateCheck, len(gates))
		copy(advisory, gates)
		for i := range advisory {
			advisory[i].Description = advisory[i].Description + " (advisory)"
			advisory[i].Check = func(*Ticket) error { return nil }
		}
		return advisory
	case RiskHigh, RiskCritical:
		// High/critical: gates are the same but enforced strictly.
		// Future: could add additional gates (extra reviewers, etc.)
		return gates
	default:
		return gates
	}
}

// Gate check implementations.

func checkDescriptionExists(t *Ticket) error {
	body := strings.TrimSpace(t.Body)
	if body == "" {
		return fmt.Errorf("ticket has no description")
	}
	return nil
}

func checkACExists(t *Ticket) error {
	if !strings.Contains(t.Body, "## Acceptance Criteria") {
		return fmt.Errorf("missing ## Acceptance Criteria section")
	}
	return nil
}

func checkDesignExists(t *Ticket) error {
	if !strings.Contains(t.Body, "## Design") && !strings.Contains(t.Body, "## Implementation Plan") {
		return fmt.Errorf("missing ## Design or ## Implementation Plan section")
	}
	return nil
}

func checkReviewApproved(t *Ticket) error {
	if t.Review != ReviewApproved {
		return fmt.Errorf("review state is %q, want approved", t.Review)
	}
	return nil
}

func checkCodeReviewApproved(t *Ticket) error {
	for _, r := range t.Reviews {
		if strings.Contains(r.Reviewer, "code-review") && r.Verdict == "approved" {
			return nil
		}
	}
	return fmt.Errorf("no approved code review found")
}

func checkImplReviewApproved(t *Ticket) error {
	for _, r := range t.Reviews {
		if strings.Contains(r.Reviewer, "impl-review") && r.Verdict == "approved" {
			return nil
		}
	}
	return fmt.Errorf("no approved impl review found")
}

func checkAdvisoryReviewSurfaced(t *Ticket) error {
	// For chores: just needs any review record to exist.
	if len(t.Reviews) > 0 {
		return nil
	}
	return fmt.Errorf("no advisory review recorded")
}

func checkTestResultsRecorded(t *Ticket) error {
	if !strings.Contains(t.Body, "## Test Results") {
		return fmt.Errorf("missing ## Test Results section")
	}
	return nil
}
