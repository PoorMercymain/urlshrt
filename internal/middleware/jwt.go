package middleware

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int64
}

func GetUserId(tokenString string) int64 {
    claims := &Claims{}
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
    return claims.UserID
}

func BuildJWTString() (string, int64, error) {
	id, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		util.GetLogger().Infoln("could not generate", err)
		return "", -1, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims {
        RegisteredClaims: jwt.RegisteredClaims{}, //мб не надо
        UserID: id.Int64(),
    })

    tokenString, err := token.SignedString([]byte("ultrasecretkey"))
    if err != nil {
		util.GetLogger().Infoln("could not create token", err)
        return "", -1, err
    }

	util.GetLogger().Infoln("idd", id)

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
			util.GetLogger().Infoln("тута")
			hasCookie = true
			cookieString = cookie.String()
		} else {
			util.GetLogger().Infoln("тута1")
			cookieString = ""
		}

		var jwtStr string

		ctx := r.Context()

		if len(cookieString) > 5 {
			util.GetLogger().Infoln("jwt str", cookieString)
			cookieString = cookieString[len("auth="):]
			util.GetLogger().Infoln("jwt str1", cookieString)
		}

		if id = GetUserId(cookieString); id == -1 || !hasCookie {
			//создаем новую куку
			//надо будет передавать через response в хэндлере
			util.GetLogger().Infoln("здеся")

			jwtStr, id, err = BuildJWTString()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Header.Set("Cookie", "auth=" + jwtStr)
			ctx = context.WithValue(ctx, "unauthorized", true)
		}
		fmt.Println("id", id)
		ctx = context.WithValue(ctx, "id", id)

		fmt.Println(ctx.Value("id").(int64))

		r = r.WithContext(ctx)
		fmt.Println(r.Context().Value("id"))

		h.ServeHTTP(w, r)
	}

	return jwtFn
}