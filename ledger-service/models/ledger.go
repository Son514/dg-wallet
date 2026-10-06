package models

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type LedgerEntry struct {
	ID        int64
	WalletID  int64
	Amount    string
	Type      string
	CreatedAt time.Time
}

func TransactionHistory(db *sql.DB, walletID int64) ([]LedgerEntry, error) {
	rows, err := db.Query(
		"SELECT id, wallet_id, amount::text, type, created_at FROM ledger_entries WHERE wallet_id = $1 ORDER BY created_at DESC, id DESC",
		walletID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]LedgerEntry, 0)
	for rows.Next() {
		var entry LedgerEntry
		if err := rows.Scan(&entry.ID, &entry.WalletID, &entry.Amount, &entry.Type, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func CreateLedgerEntries(db *sql.DB, entries []LedgerEntry) ([]int64, error) {
	if len(entries) == 0 {
		return []int64{}, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(pq.CopyIn("ledger_entries", "wallet_id", "amount", "type"))
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if _, err := stmt.Exec(e.WalletID, e.Amount, e.Type); err != nil {
			return nil, err
		}
	}

	if _, err := stmt.Exec(); err != nil {
		return nil, err
	}

	if err := stmt.Close(); err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(entries))
	rows, err := tx.Query("SELECT id FROM ledger_entries ORDER BY id DESC LIMIT $1", len(entries))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// The query above fetched ids newest-first; reverse them so the
	// response lists entry ids in the same order as the input entries.
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}

	return ids, nil
}
