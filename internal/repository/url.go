package repository

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type url struct {
	location string
}

func NewURL(location string) *url {
	return &url{location: location}
}

func (r *url) PingPg(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()
	pg, err := state.GetPgPtr()
	if err != nil {
		return err
	}
    err = pg.PingContext(ctx)
	return err
}

func (r *url) ReadAll(ctx context.Context) ([]state.URLStringJSON, error) {
	var db *sql.DB
	var err error
	if db, err = state.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || state.GetDSN() == "" {
		f, err := os.Open(r.location)
		util.GetLogger().Infoln("got")
		if err != nil {
			util.GetLogger().Infoln("get", err)
			return nil, err
		}

		defer func() error {
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		}()

		scanner := bufio.NewScanner(f)

		jsonSlice := make([]state.URLStringJSON, 0)
		var jsonSliceElemBuffer state.URLStringJSON

		for scanner.Scan() {
			buf := bytes.NewBuffer([]byte(scanner.Text()))

			err := json.Unmarshal(buf.Bytes(), &jsonSliceElemBuffer)
			if err != nil {
				return nil, err
			}

			jsonSlice = append(jsonSlice, jsonSliceElemBuffer)
		}

		return jsonSlice, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT * FROM urlshrt")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	urlsFromPg := make([]state.URLStringJSON, 0)
	for rows.Next() {
		var u state.URLStringJSON

		err = rows.Scan(&u.UUID, &u.ShortURL, &u.OriginalURL)
		if err != nil {
			return nil, err
		}
		urlsFromPg = append(urlsFromPg, u)
	}
	return urlsFromPg, nil
}

func (r *url) Create(ctx context.Context, urls []state.URLStringJSON) error {
	var db *sql.DB
	var err error
	if db, err = state.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || state.GetDSN() == "" {
		if r.location == "" {
			return nil
		}
		err := os.MkdirAll(filepath.Dir(r.location), 0600)
		if err != nil {
			util.GetLogger().Infoln("save mkdir", err)
			return err
		}

		f, err := os.OpenFile(r.location, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			util.GetLogger().Infoln("save", err)
			return err
		}

		defer func() error {
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		}()

		for _, str := range urls {
			jsonByteSlice, err := json.Marshal(str)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(jsonByteSlice)
			buf.WriteByte('\n')
			f.WriteString(buf.String())
		}

		return nil
	}
	for _, url := range urls {
		_, err = db.ExecContext(ctx, "INSERT INTO urlshrt VALUES($1, $2, $3)", url.UUID, url.ShortURL, url.OriginalURL)
	}
	return err
}
