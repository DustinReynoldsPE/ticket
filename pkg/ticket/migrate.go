package ticket

// NeedsMigration returns true if the ticket has a status but no stage.
func NeedsMigration(t *Ticket) bool {
	return t.Stage == "" && t.Status != ""
}

// MigrateTicket sets the stage from the status if missing. Returns true if a
// change was made. Status is preserved for backward compatibility.
func MigrateTicket(t *Ticket) bool {
	if !NeedsMigration(t) {
		return false
	}
	if stage, ok := legacyStatusToStage[string(t.Status)]; ok {
		t.Stage = stage
	} else {
		t.Stage = StageTriage
	}
	return true
}

// MigrateAll migrates all legacy tickets in the store. Returns the count of
// tickets that were migrated.
func MigrateAll(store *FileStore) (int, error) {
	tickets, err := store.List()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range tickets {
		if MigrateTicket(t) {
			if err := store.Update(t); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}
