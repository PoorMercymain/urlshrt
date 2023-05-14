package domain

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"strings"
)

type Database struct {
	dBType   string
	location string
}

func NewDB(dBType string, location string) *Database {
	return &Database{dBType: dBType, location: location}
}

func (db *Database) getUrls() ([]URL, error) {
	f, err := os.Open(db.location)
	if err != nil {
		return make([]URL, 0), err
	}

	defer func() error {
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	}()

	scanner := bufio.NewScanner(f)

	urls := make([]URL, 0)

	for scanner.Scan() {
		scannedTextSlice := strings.Split(scanner.Text(), " ")

		if len(scannedTextSlice) != 2 {
			return make([]URL, 0), errors.New("incorrect database! It should have 2 elements with a whitespace between them in any string! ")
		}
		u := URL{Original: strings.Split(scanner.Text(), " ")[0], Shortened: strings.Split(scanner.Text(), " ")[1]}
		urls = append(urls, u)
	}

	return urls, nil
}

func (db *Database) saveStrings(urlStrings []string) error {
	f, err := os.OpenFile(db.location, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer func() error {
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	}()

	for _, str := range urlStrings {
		buf := bytes.NewBuffer([]byte(str))
		buf.WriteByte('\n')
		f.WriteString(buf.String())
	}

	return nil
}
