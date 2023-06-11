package state

import (
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var pg *sql.DB
var dsn string

func ConnectToPG(DSN string) error {
	var err error
	pg, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	_, err = pg.Exec("CREATE TABLE IF NOT EXISTS urlshrt(uuid INTEGER, short text, original text)")
	dsn = DSN
	return err
}

func GetPgPtr() (*sql.DB, error) {
	if pg != nil {
		return pg, nil
	}
	return pg, errors.New("postgres was not initialized")
}

func GetDSN() string {
	return dsn
}