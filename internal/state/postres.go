package state

import (
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var pg *sql.DB

func ConnectToPG(DSN string) error {
	var err error
	pg, err = sql.Open("pgx", DSN)
	return err
}

func GetPgPtr() (*sql.DB, error) {
	if pg != nil {
		return pg, nil
	}
	return pg, errors.New("postgres was not initialized")
}