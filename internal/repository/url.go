package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type url struct {
	location string
}

func NewURL(location string) *url {
	return &url{location: location}
}

func (r *url) ReadAll(ctx context.Context) ([]state.URLStringJSON, error) {
	f, err := os.Open(r.location)
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

func (r *url) Create(ctx context.Context, urls []state.URLStringJSON) error {
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
