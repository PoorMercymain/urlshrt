package middleware

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/golang-jwt/jwt/v4"
)

func GetUserID(tokenString string) int64 {
	claims := jwt.MapClaims{
		"userid": int64(-1),
	}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte("ultrasecretkey"), nil
	})

	if err != nil {
		util.GetLogger().Infoln("Couldn't parse", err)
		return -1
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return -1
	}

	fmt.Println("Token is valid")
	util.GetLogger().Infoln(claims["userid"])
	uid := int64(claims["userid"].(float64))
	return uid
}

func BuildJWTString() (string, int64, error) {
	id, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		util.GetLogger().Infoln("could not generate", err)
		return "", -1, err
	}

	claims := jwt.MapClaims{
		"userid": id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("ultrasecretkey"))
	if err != nil {
		util.GetLogger().Infoln("could not create token", err)
		return "", -1, err
	}

	util.GetLogger().Infoln("id2", id)

	return tokenString, id.Int64(), nil
}

func Authorize(h http.Handler) http.HandlerFunc {
	jwtFn := func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth")
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var id int64
		var hasCookie bool
		var cookieString string
		if !errors.Is(err, http.ErrNoCookie) {
			hasCookie = true
			cookieString = cookie.String()
		}

		var jwtStr string

		ctx := r.Context()

		if len(cookieString) > len("auth=") {
			jwtStr = cookieString[len("auth="):]
		}

		if id = GetUserID(jwtStr); id == -1 || !hasCookie {
			jwtStr, id, err = BuildJWTString()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cookieToSend := http.Cookie{Name: "auth", Value: jwtStr}
			http.SetCookie(w, &cookieToSend)
			ctx = context.WithValue(ctx, domain.Key("unauthorized"), true)
		}

		util.GetLogger().Infoln("id", id)
		ctx = context.WithValue(ctx, domain.Key("id"), id)

		util.GetLogger().Infoln(ctx.Value(domain.Key("id")).(int64))

		r = r.WithContext(ctx)
		util.GetLogger().Infoln(r.Context().Value(domain.Key("id")))

		h.ServeHTTP(w, r)
	}

	return jwtFn
}
