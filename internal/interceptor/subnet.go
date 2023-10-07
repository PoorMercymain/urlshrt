package interceptor

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func CheckCIDR(CIDR string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		const statsMethodName = "/api.v1.UrlshrtV1/ReadAmountOfURLsAndUsersV1"

		if info.FullMethod == statsMethodName {
			if CIDR == "" {
				return nil, status.Error(codes.PermissionDenied, "Forbidden")
			}

			pr, ok := peer.FromContext(ctx)
			if !ok {
				return nil, status.Error(codes.Internal, "Failed to get peer from context")
			}

			_, subnet, err := net.ParseCIDR(CIDR)
			if err != nil {
				return nil, status.Error(codes.Internal, "Failed to parse CIDR")
			}

			host, _, err := net.SplitHostPort(pr.Addr.String())
			if err != nil { // that may happen if there are no port in address
				host = pr.Addr.String()
			}

			parsedIP := net.ParseIP(host)
			if parsedIP == nil {
				return nil, status.Error(codes.Internal, "Failed to parse IP")
			}

			if !subnet.Contains(parsedIP) {
				return nil, status.Error(codes.PermissionDenied, "Forbidden")
			}
		}

		return handler(ctx, req)
	}
}
