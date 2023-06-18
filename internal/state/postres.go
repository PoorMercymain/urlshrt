package state

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PoorMercymain/urlshrt/pkg/util"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Postgres struct {
	pg *sql.DB
	dsn string
}

func NewPG(DSN string) (*Postgres, error) {
	var err error
	pg, err := sql.Open("pgx", DSN)
	if err != nil {
		return nil, err
	}
	err = goose.SetDialect("postgres")
	if err != nil {
		return nil, err
	}

	err = pg.PingContext(context.Background())
	if err != nil {
		return &Postgres{}, err
	}

	err = goose.Run("up", pg, "./pkg/migrations")
	if err != nil {
		util.GetLogger().Infoln(err)
		return nil, err
	}

	dsn := DSN
	return &Postgres{pg: pg, dsn: dsn}, err
}

func(s *Postgres) GetPgPtr() (*sql.DB, error) {
	if s.pg != nil {
		return s.pg, nil
	}
	return s.pg, errors.New("postgres was not initialized")
}

func(s *Postgres) GetDSN() string {
	return s.dsn
}