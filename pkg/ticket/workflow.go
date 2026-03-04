package ticket

import (
	"fmt"
	"time"
)

// AdvanceOptions configures how a ticket advances through the pipeline.
type AdvanceOptions struct {
	SkipTo Stage  // If set, skip directly to this stage instead of next.
	Reason string // Required when skipping stages.
	Force  bool   // Bypass gate checks.
}

// AdvanceResult describes what happened during a stage advance.
type AdvanceResult struct {
	From       Stage
	To         Stage
	Skipped    []Stage // Stages that were skipped (if SkipTo was used).
	GateErrors []error // Gate failures (empty on success or force).
}

// Advance moves a ticket to its next pipeline stage, enforcing gate checks.
// The ticket is persisted to the store on success.
func Advance(store *FileStore, id string, opts AdvanceOptions) (*AdvanceResult, error) {
	t, err := store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("ticket %s: %w", id, err)
	}

	if t.Stage == "" {
		return nil, fmt.Errorf("ticket %s has no stage — migrate it first", id)
	}

	pipeline, err := PipelineFor(t.Type)
	if err != nil {
		return nil, err
	}

	from := t.Stage
	var to Stage
	var skipped []Stage

	if opts.SkipTo != "" {
		// Validate that SkipTo is a valid stage for this type and is ahead.
		toIdx := StageIndex(t.Type, opts.SkipTo)
		fromIdx := StageIndex(t.Type, from)
		if toIdx < 0 {
			return nil, fmt.Errorf("stage %q is not part of the %s pipeline", opts.SkipTo, t.Type)
		}
		if toIdx <= fromIdx {
			return nil, fmt.Errorf("cannot skip backward from %s to %s", from, opts.SkipTo)
		}
		to = opts.SkipTo
		// Collect skipped stages.
		for i := fromIdx + 1; i < toIdx; i++ {
			skipped = append(skipped, pipeline[i])
		}
		if opts.Reason == "" && len(skipped) > 0 {
			return nil, fmt.Errorf("reason is required when skipping stages")
		}
	} else {
		next, ok := NextStage(t.Type, from)
		if !ok {
			return nil, fmt.Errorf("ticket %s is already at final stage %s", id, from)
		}
		to = next
	}

	// Run gate checks.
	result := &AdvanceResult{From: from, To: to, Skipped: skipped}

	if !opts.Force {
		gateErrors := CheckGates(t, to)
		if len(gateErrors) > 0 {
			result.GateErrors = gateErrors
			return result, fmt.Errorf("gate checks failed for %s → %s", from, to)
		}
	}

	// Apply the advance.
	t.Stage = to
	t.Review = ReviewNone // Reset review state for new stage.
	if len(skipped) > 0 {
		t.Skipped = append(t.Skipped, skipped...)
	}

	if err := store.Update(t); err != nil {
		return nil, err
	}

	return result, nil
}

// Skip is a convenience wrapper around Advance with SkipTo set.
func Skip(store *FileStore, id string, to Stage, reason string) (*AdvanceResult, error) {
	return Advance(store, id, AdvanceOptions{
		SkipTo: to,
		Reason: reason,
	})
}

// SetReview records a review verdict on a ticket and appends a ReviewRecord.
func SetReview(store *FileStore, id string, reviewer string, verdict ReviewState, comment string) error {
	t, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("ticket %s: %w", id, err)
	}

	if err := ValidateReviewState(verdict); err != nil {
		return err
	}

	t.Review = verdict
	t.Reviews = append(t.Reviews, ReviewRecord{
		Timestamp: time.Now().UTC(),
		Reviewer:  reviewer,
		Verdict:   string(verdict),
		Comment:   comment,
		Stage:     t.Stage,
	})

	return store.Update(t)
}

// StageChange describes a stage propagation event.
type StageChange struct {
	ID       string
	OldStage Stage
	NewStage Stage
}

// PropagateStage checks all children of the given ticket's parent and
// propagates stages upward:
//   - All children at done → parent advances to done
//   - All children at test or later → parent advances to test (if in pipeline)
//
// Recurses upward through the parent chain. Returns a slice of changes made.
func PropagateStage(store *FileStore, childID string) ([]StageChange, error) {
	child, err := store.Get(childID)
	if err != nil {
		return nil, nil
	}
	if child.Parent == "" {
		return nil, nil
	}

	parent, err := store.Get(child.Parent)
	if err != nil {
		return nil, nil
	}
	if parent.Stage == "" {
		return nil, nil
	}

	tickets, err := store.List()
	if err != nil {
		return nil, err
	}

	allDone := true
	allTestOrLater := true
	for _, t := range tickets {
		if t.Parent != parent.ID {
			continue
		}
		if t.Stage == "" {
			// Mixed legacy/pipeline children — skip propagation.
			return nil, nil
		}
		if t.Stage != StageDone {
			allDone = false
		}
		idx := StageIndex(t.Type, t.Stage)
		testIdx := StageIndex(t.Type, StageTest)
		if testIdx >= 0 && idx < testIdx {
			allTestOrLater = false
		}
	}

	var changes []StageChange

	if allDone && parent.Stage != StageDone {
		changes = append(changes, StageChange{
			ID:       parent.ID,
			OldStage: parent.Stage,
			NewStage: StageDone,
		})
		parent.Stage = StageDone
		if err := store.Update(parent); err != nil {
			return changes, err
		}
		more, err := PropagateStage(store, parent.ID)
		changes = append(changes, more...)
		if err != nil {
			return changes, err
		}
	} else if allTestOrLater && HasStage(parent.Type, StageTest) {
		testIdx := StageIndex(parent.Type, StageTest)
		parentIdx := StageIndex(parent.Type, parent.Stage)
		if parentIdx < testIdx {
			changes = append(changes, StageChange{
				ID:       parent.ID,
				OldStage: parent.Stage,
				NewStage: StageTest,
			})
			parent.Stage = StageTest
			if err := store.Update(parent); err != nil {
				return changes, err
			}
			more, err := PropagateStage(store, parent.ID)
			changes = append(changes, more...)
			if err != nil {
				return changes, err
			}
		}
	}

	return changes, nil
}

// Claim sets the assignee on a ticket with enforcement. If the ticket is
// already assigned to a different identity, it returns an error unless force
// is true. Claiming a ticket you already own is a no-op.
func Claim(store *FileStore, id, assignee string, force bool) error {
	if assignee == "" {
		return fmt.Errorf("assignee is required")
	}

	t, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("ticket %s: %w", id, err)
	}

	if t.Assignee != "" && t.Assignee != assignee {
		if !force {
			return fmt.Errorf("ticket %s is already assigned to %q (use --force to override)", t.ID, t.Assignee)
		}
	}

	if t.Assignee == assignee {
		return nil
	}

	t.Assignee = assignee
	return store.Update(t)
}

