package handler

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/domain/mocks"
	"github.com/PoorMercymain/urlshrt/internal/interceptor"
	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/service"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/api"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

func TestGRPC(t *testing.T) {
	err := util.InitLogger()
	require.NoError(t, err)

	state.InitShortAddress("addr")

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.Log, interceptor.Authorize("abc"),
		interceptor.CheckCIDR("127.0.0.1/32"), interceptor.ValidateRequest))
	var wg sync.WaitGroup
	var once sync.Once

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	urls := make(map[string]state.URLStringJSON)

	urls["abc"] = state.URLStringJSON{
		ShortURL:    "cba",
		OriginalURL: "abc",
		UUID:        1,
	}

	state.InitCurrentURLs(&urls)

	ur := mocks.NewMockURLRepository(ctrl)
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(false, nil).MaxTimes(1)
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(true, nil).MaxTimes(1)
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(false, errors.New("")).MaxTimes(1)
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(true, nil).MaxTimes(2)

	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", nil).MaxTimes(2)
	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", domain.NewUniqueError(errors.New(""))).MaxTimes(1)
	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", errors.New("")).MaxTimes(1)

	ur.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).Return(errors.New("")).MaxTimes(1)
	ur.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(2)

	ur.EXPECT().ReadUserURLs(gomock.Any()).Return([]state.URLStringJSON{}, nil).MaxTimes(1)
	ur.EXPECT().ReadUserURLs(gomock.Any()).Return([]state.URLStringJSON{}, errors.New("")).MaxTimes(1)

	ur.EXPECT().CountURLsAndUsers(gomock.Any()).Return(1, 1, nil).MaxTimes(1)
	ur.EXPECT().CountURLsAndUsers(gomock.Any()).Return(0, 0, errors.New("")).MaxTimes(1)

	ur.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MaxTimes(2)

	us := service.NewURL(ur)

	ch := make(chan domain.URLWithID)
	mc := domain.NewMutexChanString(ch)

	urlshrt := &Server{
		Wg:            &wg,
		Once:          &once,
		Srv:           us,
		ShortURLsChan: mc,
	}
	api.RegisterUrlshrtV1Server(grpcServer, urlshrt)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	go func() {
		errServe := grpcServer.Serve(listener)
		require.NoError(t, errServe)
	}()
	defer grpcServer.Stop()

	creds := insecure.NewCredentials()
	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	defer conn.Close()

	client := api.NewUrlshrtV1Client(conn)

	testTableReadOriginal := []struct {
		input      *api.ReadOriginalRequestV1
		statusCode codes.Code
	}{
		{&api.ReadOriginalRequestV1{Shortened: "cba"}, codes.OK},
		{&api.ReadOriginalRequestV1{Shortened: "cba"}, codes.NotFound},
		{&api.ReadOriginalRequestV1{Shortened: "cb"}, codes.InvalidArgument},
	}

	for _, test := range testTableReadOriginal {
		_, err := client.ReadOriginalV1(context.Background(), test.input)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}

	testTableCreateShortened := []struct {
		input      *api.CreateShortenedRequestV1
		statusCode codes.Code
		randSeed   string
	}{
		{&api.CreateShortenedRequestV1{Original: "cba"}, codes.OK, ""},
		{&api.CreateShortenedRequestV1{Original: "cba"}, codes.InvalidArgument, "ab"},
		{&api.CreateShortenedRequestV1{Original: "cba"}, codes.OK, "0"},
		{&api.CreateShortenedRequestV1{Original: "cba"}, codes.AlreadyExists, ""},
		{&api.CreateShortenedRequestV1{Original: "cba"}, codes.Internal, ""},
	}

	for _, test := range testTableCreateShortened {
		ctx := context.Background()
		if test.randSeed != "" {
			md := metadata.Pairs("RandSeed", test.randSeed)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		_, err := client.CreateShortenedV1(ctx, test.input)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}

	testTableCreateShortenedFromBatch := []struct {
		input      *api.CreateShortenedFromBatchRequestV1
		statusCode codes.Code
		randSeed   string
	}{
		{&api.CreateShortenedFromBatchRequestV1{Original: []*api.OriginalWithCorrelationV1{{Original: "cba", Correlation: "123"}}}, codes.OK, ""},
		{&api.CreateShortenedFromBatchRequestV1{Original: []*api.OriginalWithCorrelationV1{{Original: "c", Correlation: "123"}}}, codes.Internal, ""},
		{&api.CreateShortenedFromBatchRequestV1{Original: []*api.OriginalWithCorrelationV1{{Original: "b", Correlation: "123"}}}, codes.OK, ""},
		{&api.CreateShortenedFromBatchRequestV1{Original: []*api.OriginalWithCorrelationV1{{Original: "a", Correlation: "123"}}}, codes.InvalidArgument, "a"},
		{&api.CreateShortenedFromBatchRequestV1{Original: []*api.OriginalWithCorrelationV1{{Original: "a", Correlation: "123"}}}, codes.OK, "0"},
	}

	for _, test := range testTableCreateShortenedFromBatch {
		ctx := context.Background()
		if test.randSeed != "" {
			md := metadata.Pairs("RandSeed", test.randSeed)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		_, err := client.CreateShortenedFromBatchV1(ctx, test.input)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}

	jwt, _, err := middleware.BuildJWTString("abc")
	require.NoError(t, err)

	testTableReadUserURLs := []struct {
		statusCode codes.Code
		jwt        string
	}{
		{codes.OK, jwt},
		{codes.Unauthenticated, ""},
		{codes.InvalidArgument, jwt},
	}

	for _, test := range testTableReadUserURLs {
		ctx := context.Background()
		if test.jwt != "" {
			md := metadata.Pairs("auth", test.jwt)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
		_, err := client.ReadUserURLsV1(ctx, &emptypb.Empty{})
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}

	testTableReadAmountOfURLsAndUsers := []struct {
		statusCode codes.Code
	}{
		{codes.OK},
		{codes.Internal},
	}

	for _, test := range testTableReadAmountOfURLsAndUsers {
		_, err := client.ReadAmountOfURLsAndUsersV1(context.Background(), &emptypb.Empty{})
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}

	testTableDeleteUserURLs := []struct {
		input      *api.DeleteUserURLsRequestV1
		statusCode codes.Code
	}{
		{&api.DeleteUserURLsRequestV1{UrlsToDelete: []string{"a"}}, codes.OK},
		{&api.DeleteUserURLsRequestV1{UrlsToDelete: []string{}}, codes.InvalidArgument},
	}

	for _, test := range testTableDeleteUserURLs {
		_, err := client.DeleteUserURLsV1(context.Background(), test.input)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, test.statusCode, s.Code())
	}
}
