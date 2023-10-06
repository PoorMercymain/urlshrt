package interceptor

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
)

func GetUserID(tokenString string) (int64, error) {
	claims := jwt.MapClaims{
		"userid": int64(0),
	}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("wrong signing method")
		}
		return []byte("ultrasecretkey"), nil // TODO move key to a safer place
	})

	if err != nil {
		util.GetLogger().Infoln("Couldn't parse", err, tokenString)
		return 0, err
	}

	if !token.Valid {
		util.GetLogger().Infoln("Token is not valid")
		return 0, err
	}

	util.GetLogger().Infoln("Token is valid")
	util.GetLogger().Infoln(claims["userid"])
	uid := int64(claims["userid"].(float64))

	return uid, nil
}

func BuildJWTString() (string, int64, error) {
	id, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		util.GetLogger().Infoln("could not generate", err)
		return "", 0, err
	}

	claims := jwt.MapClaims{
		"userid": id.Int64(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("ultrasecretkey")) // TODO the same problem with key
	if err != nil {
		util.GetLogger().Infoln("could not create token", err)
		return "", 0, err
	}

	util.GetLogger().Infoln("id2", id)

	return tokenString, id.Int64(), nil
}

func Authorize(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var needToCreateJWT bool
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		needToCreateJWT = true
		util.GetLogger().Infoln("get metadata")
	}

	var values []string
	if !needToCreateJWT {
		values, ok = md["auth"]
		if !ok {
			needToCreateJWT = true
		}
	}

	var auth string
	if !needToCreateJWT {
		auth = values[0]
		if auth == "" {
			needToCreateJWT = true
		}
	}

	var uid int64
	var err error
	if !needToCreateJWT {
		uid, err = GetUserID(auth)
		if err != nil {
			needToCreateJWT = true
		}
	}

	if needToCreateJWT {
		auth, uid, err = BuildJWTString()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to build auth string")
		}

		ctx = context.WithValue(ctx, domain.Key("unauthorized"), true)
	}

	util.GetLogger().Infoln("id", uid)
	ctx = context.WithValue(ctx, domain.Key("id"), uid)

	md = metadata.Pairs("auth", auth)
	err = grpc.SendHeader(ctx, md)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send metadata back to client")
	}

	return handler(ctx, req)
}
