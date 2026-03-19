package ticket

// MigrateAll re-parses and re-writes all tickets in the store. Legacy tickets
// with a status field but no stage are auto-migrated during Parse(), and the
// status field is dropped during Serialize(). Returns the count of tickets
// that were re-written.
func MigrateAll(store *FileStore) (int, error) {
	tickets, err := store.List()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range tickets {
		if err := store.Update(t); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
