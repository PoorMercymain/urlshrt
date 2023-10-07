package handler

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"google.golang.org/protobuf/types/known/emptypb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/api"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type Server struct {
	Wg            *sync.WaitGroup
	Once          *sync.Once
	Srv           domain.URLService
	ShortURLsChan *domain.MutexChanString
	api.UnimplementedUrlshrtV1Server
}

func (h *Server) ReadOriginalV1(ctx context.Context, req *api.ReadOriginalRequestV1) (*api.ReadOriginalReplyV1, error) {
	errChan := make(chan error, 1)
	orig, err := h.Srv.ReadOriginal(ctx, req.Shortened, errChan)
	select {
	case <-errChan: // if url was deleted, a message in errChan shall appear
		return nil, status.Errorf(codes.NotFound, "requested URL is deleted from the service")
	default:
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "error in request or the shortened url does not exist")
		}
	}

	return &api.ReadOriginalReplyV1{Original: orig}, nil
}

func (h *Server) CreateShortenedV1(ctx context.Context, req *api.CreateShortenedRequestV1) (*api.CreateShortenedReplyV1, error) {
	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "couldn't get metadata from context")
	}

	randSeedValues := md.Get("RandSeed")
	var randSeed int
	var err error
	if len(randSeedValues) > 0 {
		randSeed, err = strconv.Atoi(randSeedValues[0])
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "incorrect format of RandSeed used")
		}

		ctx = context.WithValue(ctx, domain.Key("seed"), int64(randSeed))
	}

	shortenedURL, err := h.Srv.CreateShortened(ctx, req.Original)
	var uErr *domain.UniqueError
	if err != nil && errors.As(err, &uErr) {
		return &api.CreateShortenedReplyV1{Shortened: addr + shortenedURL},
			status.Errorf(codes.AlreadyExists, "provided URL already exist in the service")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "something went wrong in the service")
	}

	return &api.CreateShortenedReplyV1{Shortened: addr + shortenedURL}, nil
}

func (h *Server) CreateShortenedFromBatchV1(ctx context.Context, req *api.CreateShortenedFromBatchRequestV1) (*api.CreateShortenedFromBatchReplyV1, error) {
	batch := make([]*domain.BatchElement, len(req.Original))
	for i, elem := range req.Original {
		batch[i] = &domain.BatchElement{ID: elem.Correlation, OriginalURL: elem.Original}
	}

	util.GetLogger().Infoln(batch)

	shortened, err := h.Srv.CreateShortenedFromBatch(ctx, batch, h.Wg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "something went wrong while processing the request")
	}

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	shortenedResult := &api.CreateShortenedFromBatchReplyV1{}
	shortenedResult.Shortened = make([]*api.ShortenedWithCorrelationV1, len(shortened))
	for i, shrt := range shortened {
		shortenedResult.Shortened[i] = &api.ShortenedWithCorrelationV1{Correlation: shrt.ID, Shortened: addr + shrt.ShortenedURL}
	}

	return shortenedResult, nil
}

func (h *Server) ReadUserURLsV1(ctx context.Context, req *emptypb.Empty) (*api.ReadUserURLsReplyV1, error) {
	UserURLs, err := h.Srv.ReadUserURLs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request might be incorrect")
	}

	if unauthorized := ctx.Value(domain.Key("unauthorized")); unauthorized != nil {
		return nil, status.Errorf(codes.Unauthenticated, "please use jwt from response metadata to access the handler")
	}

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	userURLsReply := &api.ReadUserURLsReplyV1{OriginalWithShortened: make([]*api.OriginalWithShortenedV1, len(UserURLs))}
	for i, url := range UserURLs {
		userURLsReply.OriginalWithShortened[i] = &api.OriginalWithShortenedV1{Original: url.OriginalURL, Shortened: addr + url.ShortURL}
	}

	return userURLsReply, nil
}

func (h *Server) ReadAmountOfURLsAndUsersV1(ctx context.Context, req *emptypb.Empty) (*api.ReadAmountOfURLsAndUsersReplyV1, error) {
	readAmountReply := &api.ReadAmountOfURLsAndUsersReplyV1{}

	var err error
	var urlsAmount int
	var usersAmount int
	urlsAmount, usersAmount, err = h.Srv.CountURLsAndUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "something went wrong while processing the request")
	}

	readAmountReply.UrlsAmount = int64(urlsAmount)
	readAmountReply.UsersAmount = int64(usersAmount)

	return readAmountReply, nil
}

func (h *Server) DeleteUserURLsV1(ctx context.Context, req *api.DeleteUserURLsRequestV1) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, domain.Key("id"), int64(1))
	shortURLWithID := make([]domain.URLWithID, len(req.UrlsToDelete))
	for i, url := range req.UrlsToDelete {
		shortURLWithID[i] = domain.URLWithID{ID: ctx.Value(domain.Key("id")).(int64), URL: url}
	}

	go func() {
		h.Srv.DeleteUserURLs(ctx, shortURLWithID, h.ShortURLsChan, h.Once, h.Wg)
	}()

	return &emptypb.Empty{}, nil
}
