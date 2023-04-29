package domain

import (
	"bufio"
	"os"
	"strings"
)

type DB struct {
	dBType   string
	location string
}

func NewDB(dBType string, location string) DB {
	return DB{dBType: dBType, location: location}
}

func (db DB) getUrls() ([]URL, error) {
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
		u := URL{Original: strings.Split(scanner.Text(), " ")[0], Shortened: strings.Split(scanner.Text(), " ")[1]}
		urls = append(urls, u)
	}

	return urls, nil
}

func (db DB) saveStrings(urlStrings []string) error {
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
		f.WriteString(str + "\n")
	}

	return nil
}