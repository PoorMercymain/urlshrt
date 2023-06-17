package state

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var pg *sql.DB
var dsn string

func ConnectToPG(DSN string) error {
	var err error
	pg, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	err = goose.SetDialect("postgres")
	if err != nil {
		return err
	}


	err = goose.Run("up", pg, "./pkg/migrations")
	if err != nil {
		curDir, errCurDir := os.Getwd()
		fmt.Println("\nhere", curDir, "err", errCurDir)
		return err
	}

	//_, err = pg.Exec("CREATE TABLE IF NOT EXISTS urlshrt(uuid INTEGER, short text, original text primary key)")
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