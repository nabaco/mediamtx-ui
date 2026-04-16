package db

import (
	"database/sql"
	"fmt"
)

type StreamMeta struct {
	PathName    string
	Description string
}

func UpsertStreamMeta(db *sql.DB, pathName, description string) error {
	_, err := db.Exec(
		`INSERT INTO stream_metadata (path_name, description)
		 VALUES (?, ?)
		 ON CONFLICT(path_name) DO UPDATE SET description=excluded.description`,
		pathName, description,
	)
	return err
}

func GetStreamMeta(db *sql.DB, pathName string) (*StreamMeta, error) {
	var m StreamMeta
	err := db.QueryRow(`SELECT path_name, description FROM stream_metadata WHERE path_name = ?`, pathName).
		Scan(&m.PathName, &m.Description)
	if err == sql.ErrNoRows {
		return &StreamMeta{PathName: pathName}, nil
	}
	return &m, err
}

func DeleteStreamMeta(db *sql.DB, pathName string) error {
	_, err := db.Exec(`DELETE FROM stream_metadata WHERE path_name = ?`, pathName)
	return fmt.Errorf("delete stream meta: %w", err)
}

func ListStreamMeta(db *sql.DB) (map[string]*StreamMeta, error) {
	rows, err := db.Query(`SELECT path_name, description FROM stream_metadata`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]*StreamMeta)
	for rows.Next() {
		var m StreamMeta
		if err := rows.Scan(&m.PathName, &m.Description); err != nil {
			return nil, err
		}
		out[m.PathName] = &m
	}
	return out, rows.Err()
}
