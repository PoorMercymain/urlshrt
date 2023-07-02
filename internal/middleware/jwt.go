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

type Claims struct {
	jwt.RegisteredClaims
	UserID int64 `json:"uid,omitempty"`
}

func GetUserID(tokenString string) int64 {
    //claims := &Claims{}
	claims := jwt.MapClaims{
		"userid": int64(-1),
	}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
        return []byte("ultrasecretkey"), nil
    })
	util.GetLogger().Infoln(tokenString)
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
			//util.GetLogger().Infoln("тута")
			hasCookie = true
			util.GetLogger().Infoln(cookie)
			cookieString = cookie.String()
		} else {
			//util.GetLogger().Infoln("тута1")
			cookieString = ""
		}

		var jwtStr string

		ctx := r.Context()

		util.GetLogger().Infoln(cookieString)

		if len(cookieString) > 5 {
			//util.GetLogger().Infoln("jwt str", cookieString)
			cookieString = cookieString[len("auth="):]
			//util.GetLogger().Infoln("jwt str1", cookieString)
		}

		if id = GetUserID(cookieString); id == -1 || !hasCookie {
			//создаем новую куку
			//надо будет передавать через response в хэндлере
			//util.GetLogger().Infoln("здеся")

			jwtStr, id, err = BuildJWTString()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Header.Set("Cookie", "auth=" + jwtStr)
			cookieToSend := http.Cookie{Name: "auth", Value: jwtStr}
			http.SetCookie(w, &cookieToSend)
			ctx = context.WithValue(ctx, domain.Key("unauthorized"), true)
		}
		fmt.Println("id", id)
		ctx = context.WithValue(ctx, domain.Key("id"), id)

		fmt.Println(ctx.Value(domain.Key("id")).(int64))

		r = r.WithContext(ctx)
		fmt.Println(r.Context().Value(domain.Key("id")))

		h.ServeHTTP(w, r)
	}

	return jwtFn
}