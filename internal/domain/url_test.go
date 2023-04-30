package domain

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenURLHandler(t *testing.T) {
	type args struct {
		w httptest.ResponseRecorder
		r *http.Request
		wantStatus int
	}
	tests := []struct {
		name string
		u    URL
		args []args

	}{
		{
			name: "test ya.ru",
			u: URL{},
			args: []args{
					args{
						w: *httptest.NewRecorder(),
						r: func () *http.Request {
							req, _ :=  http.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader("https://ya.ru"))
							return req
						}(),
						wantStatus: http.StatusCreated,
					},
					args{
						w: *httptest.NewRecorder(),
						r: nil,
						wantStatus: http.StatusTemporaryRedirect,
					},
				},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args[0].r.Header.Set("Content-Type", "text/plain")
			urls := make([]URL, 0)
			urls = append(urls, URL{Original: "https://ya.ru", Shortened: "aBcDeFg"})
			tt.u.generateShortURL(&tt.args[0].w, tt.args[0].r, urls)

			res := tt.args[0].w.Result()
			defer res.Body.Close()
			t.Log(urls)
			assert.Equal(t, http.StatusCreated, res.StatusCode)
			resBody, err := io.ReadAll(res.Body)
			t.Log(string(resBody))
			require.NoError(t, err)
			tt.args[1].r, _ = http.NewRequest(http.MethodGet, strings.TrimPrefix(string(resBody), "http://localhost:8080") , nil)
			tt.u.getOriginalURL(&tt.args[1].w, tt.args[1].r, urls)
			getRes := tt.args[1].w.Result()
			defer getRes.Body.Close()
			getResBody, _ := io.ReadAll(getRes.Body)

			if !assert.Equal(t, http.StatusTemporaryRedirect, getRes.StatusCode) {
				t.Log(getResBody)
			}
			assert.Equal(t, "https://ya.ru", getRes.Header.Get("Location"))
		})
	}
}
