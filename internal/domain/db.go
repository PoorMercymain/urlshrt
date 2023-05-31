package domain

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

type URLStringJSON struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Database struct {
	dBType   string
	location string
}

func NewDB(dBType string, location string) *Database {
	return &Database{dBType: dBType, location: location}
}

func (db *Database) GetUrls() ([]URLStringJSON, error) {
	f, err := os.Open(db.location)
	if err != nil {
		GetLogger().Infoln("get", err)
		return nil, err
	}

	defer func() error {
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	}()

	scanner := bufio.NewScanner(f)

	jsonSlice := make([]URLStringJSON, 0)
	var jsonSliceElemBuffer URLStringJSON

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

func (db *Database) saveStrings(urls []URLStringJSON) error {
	err := os.MkdirAll(filepath.Dir(db.location), 0600)
	if err != nil {
		GetLogger().Infoln("save mkdir", err)
		return err
	}

	f, err := os.OpenFile(db.location, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		GetLogger().Infoln("save", err)
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
