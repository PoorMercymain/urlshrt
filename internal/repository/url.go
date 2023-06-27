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
	"runtime"
	"sync"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/sync/errgroup"
)

type url struct {
	locationOfJSON string
	pg *state.Postgres
}

func NewURL(locationOfJSON string, pg *state.Postgres) *url {
	return &url{locationOfJSON: locationOfJSON, pg: pg}
}

func (r *url) PingPg(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()
	pg, err := r.pg.GetPgPtr()
	if err != nil {
		return err
	}
    err = pg.PingContext(ctx)
	return err
}

func (r *url) ReadAll(ctx context.Context) ([]state.URLStringJSON, error) {
	var db *sql.DB
	var err error
	if db, err = r.pg.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		f, err := os.Open(r.locationOfJSON)
		if err != nil {
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
	rows, err := db.QueryContext(ctx, "SELECT uuid, short, original FROM urlshrt")
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

func (r *url) Create(ctx context.Context, urls []state.URLStringJSON) (string, error) {
	var db *sql.DB
	var err error

	if db, err = r.pg.GetPgPtr(); err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		if r.locationOfJSON == "" {
			return "", nil
		}
		err := os.MkdirAll(filepath.Dir(r.locationOfJSON), 0600)
		if err != nil {
			util.GetLogger().Infoln("save mkdir", err)
			return "", err
		}

		f, err := os.OpenFile(r.locationOfJSON, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			util.GetLogger().Infoln("save", err)
			return "", err
		}

		defer func() error {
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		}()

		urlsFromFile, err := r.ReadAll(ctx)
		if err != nil {
			return "", err
		}
		urlsFromFileMap := make(map[string]state.URLStringJSON)
		for _, url := range urlsFromFile {
			urlsFromFileMap[url.OriginalURL] = url
		}

		for _, str := range urls {
			if _, ok := urlsFromFileMap[str.OriginalURL]; !ok {
				jsonByteSlice, err := json.Marshal(str)
				if err != nil {
					return "", err
				}
				buf := bytes.NewBuffer(jsonByteSlice)
				buf.WriteByte('\n')
				f.WriteString(buf.String())
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

func (r *url) CreateBatch(ctx context.Context, batch []*state.URLStringJSON) error {
	var db *sql.DB
	var err error

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
	}

	if r.pg == nil || err != nil || r.PingPg(ctx) != nil || r.pg.GetDSN() == "" {
		if r.locationOfJSON == "" {
			return nil
		}
		err := os.MkdirAll(filepath.Dir(r.locationOfJSON), 0600)
		if err != nil {
			util.GetLogger().Infoln("save mkdir", err)
			return err
		}

		f, err := os.OpenFile(r.locationOfJSON, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
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

		for _, str := range batch {
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

	tx, err := db.Begin()
    if err != nil {
        return err
    }
	defer tx.Rollback()

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

	return tx.Commit()
}

func(r *url) ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error) {
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

func(r *url) DeleteUserURLs(ctx context.Context, shortURLs []string) error {
	begin := time.Now()
	var db *sql.DB
	var err error

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
		if err != nil {
			return err
		}
	}

	shortURLsChan := make(chan string, len(shortURLs))

	statement, err := db.PrepareContext(ctx, "SELECT user_id, is_deleted FROM urlshrt WHERE short = $1")
	if err != nil {
		return err
	}
	defer statement.Close()
	inputChan := make(chan string, len(shortURLs))

	for _, url := range shortURLs {
		inputChan <-url
	}
	close(inputChan)
	var wg sync.WaitGroup
	wg.Add(len(inputChan))
	for i := 0; i < runtime.NumCPU(); i++ {
		if u, ok := <-inputChan; ok {
			u := u
			go func() {
				var userID int
				var isDeleted int
				row := statement.QueryRowContext(ctx, u)
				row.Scan(&userID, &isDeleted)
				if int64(userID) == ctx.Value(domain.Key("id")).(int64) && isDeleted == 0 {
					shortURLsChan <-u
					util.GetLogger().Infoln("положил", u)
					wg.Add(1)
				}
				wg.Done()
				if time.Since(begin) > time.Second*30 {
				util.GetLogger().Infoln("зависло1")
			}
			}()
		}
	}


	tx, err := db.Begin()
    if err != nil {
        return err
    }
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "UPDATE urlshrt SET is_deleted = 1 WHERE short = $1")

	if err != nil {
		return err
	}

	defer stmt.Close()
	g := new(errgroup.Group)

	g.Go(func() error {
		for {
			short, ok := <-shortURLsChan
			if !ok {
				return nil
			}
			_, err = stmt.ExecContext(ctx, short)
			util.GetLogger().Infoln("shrt", short)

			if err != nil {
				wg.Done()
				return err
			}
			wg.Done()
			util.GetLogger().Infoln("l e n", len(shortURLsChan))
			if len(shortURLsChan) == 0 {
				return nil
			}
			if time.Since(begin) > time.Second*30 {
				util.GetLogger().Infoln("зависло")
			}
		}
	})

	if err := g.Wait(); err != nil {
		return err
	}
	wg.Wait()
	return tx.Commit()
}

func(r *url) IsURLDeleted(ctx context.Context, shortened string) (bool, error) {
	var db *sql.DB
	var err error
	var isDeleted int

	if r.pg != nil {
		db, err = r.pg.GetPgPtr()
		if err != nil {
			return false, err
		}
	}

	row := db.QueryRowContext(ctx, "SELECT is_deleted FROM urlshrt WHERE short = $1", shortened)
	row.Scan(&isDeleted)
	if isDeleted == 0 {
		return false, nil
	}
	return true, nil
}
