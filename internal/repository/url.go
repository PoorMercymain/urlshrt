// repository package contains some functions to use on database level of the app.
package repository

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type URL struct {
	locationOfJSON string
	pg             *state.Postgres
}

func NewURL(locationOfJSON string, pg *state.Postgres) *URL {
	return &URL{locationOfJSON: locationOfJSON, pg: pg}
}

func (r *URL) WithTransaction(db *sql.DB, txFunc func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
		if err != nil {
			return
		}
	}()

	err = txFunc(tx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *URL) PingPg(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	pg, err := r.pg.GetPgPtr()
	if err != nil {
		return err
	}
	err = pg.PingContext(ctx)
	return err
}

// ReadAll is a function which is used in another function of the app's database level. It gets all URL's data from a database.
func (r *URL) ReadAll(ctx context.Context) ([]state.URLStringJSON, error) {
	var db *sql.DB
	var errOuter error
	if db, errOuter = r.pg.GetPgPtr(); errOuter != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		f, err := os.Open(r.locationOfJSON)
		if err != nil {
			return nil, err
		}

		defer func() {
			if err := f.Close(); err != nil {
				util.GetLogger().Infoln(err)
			}
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

	rows, errOuter := db.QueryContext(ctx, "SELECT uuid, short, original FROM urlshrt")
	if errOuter != nil {
		return nil, errOuter
	}
	defer rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	urlsFromPg := make([]state.URLStringJSON, 0)
	for rows.Next() {
		var u state.URLStringJSON

		errOuter = rows.Scan(&u.UUID, &u.ShortURL, &u.OriginalURL)
		if errOuter != nil {
			return nil, errOuter
		}
		urlsFromPg = append(urlsFromPg, u)
	}
	return urlsFromPg, nil
}

// Create is a function which saves the URL data (original, shortened...) to a database.
func (r *URL) Create(ctx context.Context, urls []state.URLStringJSON) (string, error) {
	var db *sql.DB
	var err error
	var f *os.File

	if db, err = r.pg.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		if r.locationOfJSON == "" {
			return "", nil
		}
		err = os.MkdirAll(filepath.Dir(r.locationOfJSON), 0600)
		if err != nil {
			util.GetLogger().Infoln("save mkdir", err)
			return "", err
		}

		f, err = os.OpenFile(r.locationOfJSON, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			util.GetLogger().Infoln("save", err)
			return "", err
		}

		defer func() {
			if err = f.Close(); err != nil {
				util.GetLogger().Infoln(err)
			}
		}()

		var urlsFromFile []state.URLStringJSON
		urlsFromFile, err = r.ReadAll(ctx)
		if err != nil {
			return "", err
		}
		urlsFromFileMap := make(map[string]state.URLStringJSON)
		for _, url := range urlsFromFile {
			urlsFromFileMap[url.OriginalURL] = url
		}

		for _, str := range urls {
			if _, ok := urlsFromFileMap[str.OriginalURL]; !ok {
				var jsonByteSlice []byte
				jsonByteSlice, err = json.Marshal(str)
				if err != nil {
					return "", err
				}
				buf := bytes.NewBuffer(jsonByteSlice)
				buf.WriteByte('\n')
				_, err = f.WriteString(buf.String())
				if err != nil {
					return "", err
				}
			}
		}

		return "", nil
	}
	for _, url := range urls {

		var pgErr *pgconn.PgError
		id := ctx.Value(domain.Key("id")).(int64)
		_, err = db.ExecContext(ctx, "INSERT INTO urlshrt VALUES($1, $2, $3, $4, $5)", url.UUID, url.ShortURL, url.OriginalURL, id, 0)
		if err != nil {
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				uErr := domain.NewUniqueError(err)
				row := db.QueryRow("SELECT short FROM urlshrt WHERE original = $1", url.OriginalURL)
				var shrt string
				errScan := row.Scan(&shrt)
				if errScan != nil {
					return "", errScan
				}
				return shrt, uErr
			}
		}

	}
	return "", nil
}

// CreateBatch is a function which saves URL data to a database when original URLs were in JSON batch.
func (r *URL) CreateBatch(ctx context.Context, batch []*state.URLStringJSON) error {
	var db *sql.DB
	var err error

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
	}

	if r.pg == nil || err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		if r.locationOfJSON == "" {
			return nil
		}
		err = os.MkdirAll(filepath.Dir(r.locationOfJSON), 0600)
		if err != nil {
			util.GetLogger().Infoln("save mkdir", err)
			return err
		}

		var f *os.File
		f, err = os.OpenFile(r.locationOfJSON, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			util.GetLogger().Infoln("save", err)
			return err
		}

		defer func() {
			if err = f.Close(); err != nil {
				util.GetLogger().Infoln(err)
			}
		}()

		for _, str := range batch {
			var jsonByteSlice []byte
			jsonByteSlice, err = json.Marshal(str)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(jsonByteSlice)
			buf.WriteByte('\n')
			_, err := f.WriteString(buf.String())
			if err != nil {
				return err
			}
		}

		return nil
	}

	return r.WithTransaction(db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO urlshrt VALUES($1, $2, $3, $4, $5)")

		if err != nil {
			return err
		}

		defer stmt.Close()

		for _, bElem := range batch {
			util.GetLogger().Infoln(*bElem)
		}
		util.GetLogger().Infoln(len(batch))

		id := ctx.Value(domain.Key("id")).(int64)

		for _, url := range batch {
			util.GetLogger().Infoln(url.OriginalURL, url.ShortURL)
			_, err = stmt.ExecContext(ctx, url.UUID, url.ShortURL, url.OriginalURL, id, 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *URL) ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error) {
	var db *sql.DB
	var err error
	if db, err = r.pg.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		if err != nil {
			return make([]state.URLStringJSON, 0), err
		} else {
			return make([]state.URLStringJSON, 0), errors.New("postgres not found")
		}
	}

	id := ctx.Value(domain.Key("id")).(int64)

	rows, err := db.QueryContext(ctx, "SELECT uuid, short, original FROM urlshrt WHERE user_id = $1", id)
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

func (r *URL) DeleteUserURLs(ctx context.Context, shortURLs []string, uid []int64) error {
	var db *sql.DB
	var err error

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
		if err != nil {
			util.GetLogger().Infoln("err1", err)
			return err
		}
	}

	util.GetLogger().Infoln(shortURLs)

	return r.WithTransaction(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("UPDATE urlshrt SET is_deleted = 1 WHERE (short, user_id) IN (SELECT unnest($1::text[]), unnest($2::int[]))")

		if err != nil {
			util.GetLogger().Infoln("err4", err)
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(shortURLs, uid)
		if err != nil {
			util.GetLogger().Infoln("err5", err)
			return err
		}

		return nil
	})
}

func (r *URL) IsURLDeleted(ctx context.Context, shortened string) (bool, error) {
	var db *sql.DB
	var err error
	var isDeleted int

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
		if err != nil {
			return false, err
		}
	}

	util.GetLogger().Infoln(shortened)
	row := db.QueryRow("SELECT is_deleted FROM urlshrt WHERE short = $1", shortened)
	util.GetLogger().Infoln(row.Err())
	err = row.Scan(&isDeleted)
	if err != nil {
		return false, err
	}
	util.GetLogger().Infoln(isDeleted)
	if isDeleted == 0 {
		return false, nil
	}
	return true, nil
}

func (r *URL) CountURLsAndUsers(ctx context.Context) (int, int, error) {
	var db *sql.DB
	var err error
	var totalURLs, totalUsers int

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
		if err != nil {
			return 0, 0, err
		}
	}

	err = db.QueryRow("SELECT (SELECT COUNT(*) FROM urlshrt WHERE is_deleted = 0) AS total_urls, (SELECT COUNT(DISTINCT user_id) FROM urlshrt WHERE is_deleted = 0) AS total_users").Scan(&totalURLs, &totalUsers)
	if err != nil {
		util.GetLogger().Infoln(err)
		return 0, 0, err
	}

	return totalURLs, totalUsers, err
}
