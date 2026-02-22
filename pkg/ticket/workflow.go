package ticket

import "fmt"

// StatusChange describes a propagation event for callers to display.
type StatusChange struct {
	ID        string
	OldStatus Status
	NewStatus Status
}

// PropagateStatus checks all children of the given ticket's parent and
// propagates status upward:
//   - All children closed → parent becomes closed
//   - All children closed or needs_testing → parent becomes needs_testing
//   - Otherwise no change
//
// Recurses upward through the parent chain. Returns a slice of changes made.
func PropagateStatus(store *FileStore, childID string) ([]StatusChange, error) {
	child, err := store.Get(childID)
	if err != nil {
		return nil, nil
	}
	if child.Parent == "" {
		return nil, nil
	}

	parent, err := store.Get(child.Parent)
	if err != nil {
		return nil, nil // parent not found — nothing to propagate
	}

	// Find all children of this parent.
	tickets, err := store.List()
	if err != nil {
		return nil, err
	}

	allClosed := true
	allTestingOrClosed := true
	for _, t := range tickets {
		if t.Parent != parent.ID {
			continue
		}
		if t.Status != StatusClosed {
			allClosed = false
			if t.Status != StatusNeedsTesting {
				allTestingOrClosed = false
			}
		}
	}

	var changes []StatusChange

	if allClosed && parent.Status != StatusClosed {
		changes = append(changes, StatusChange{
			ID:        parent.ID,
			OldStatus: parent.Status,
			NewStatus: StatusClosed,
		})
		parent.Status = StatusClosed
		if err := store.Update(parent); err != nil {
			return changes, err
		}
		// Recurse upward.
		more, err := PropagateStatus(store, parent.ID)
		changes = append(changes, more...)
		if err != nil {
			return changes, err
		}
	} else if allTestingOrClosed && parent.Status != StatusNeedsTesting && parent.Status != StatusClosed {
		changes = append(changes, StatusChange{
			ID:        parent.ID,
			OldStatus: parent.Status,
			NewStatus: StatusNeedsTesting,
		})
		parent.Status = StatusNeedsTesting
		if err := store.Update(parent); err != nil {
			return changes, err
		}
		more, err := PropagateStatus(store, parent.ID)
		changes = append(changes, more...)
		if err != nil {
			return changes, err
		}
	}

	return changes, nil
}

// SetStatus updates a ticket's status, writes it, and propagates to parent.
// Returns any propagation changes made.
func SetStatus(store *FileStore, id string, status Status) ([]StatusChange, error) {
	if err := ValidateStatus(status); err != nil {
		return nil, err
	}

	t, err := store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("ticket %s: %w", id, err)
	}

	t.Status = status
	if err := store.Update(t); err != nil {
		return nil, err
	}

	return PropagateStatus(store, id)
}
