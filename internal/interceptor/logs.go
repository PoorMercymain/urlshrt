package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/PoorMercymain/urlshrt/pkg/util"
)

func Log(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	s, _ := status.FromError(err)

	var size int
	if resp != nil {
		bytes, err := proto.Marshal(resp.(proto.Message)) // may be not really optimal, but it works
		if err == nil {
			size = len(bytes)
		}
	}

	util.GetLogger().Infoln(
		"method", info.FullMethod,
		"duration", time.Since(start),
		"status", s.Code(),
		"size", size,
	)

	return resp, err
}
