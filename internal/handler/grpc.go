package handler

import (
	"context"
	"errors"
	"strconv"
	"sync"

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
	api.UnimplementedUrlshrtServer
}

func (h *Server) ReadOriginal(ctx context.Context, req *api.ReadOriginalRequest) (*api.ReadOriginalReply, error) {
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

	return &api.ReadOriginalReply{Original: orig}, nil
}

func (h *Server) CreateShortened(ctx context.Context, req *api.CreateShortenedRequest) (*api.CreateShortenedReply, error) {
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

	// TODO use user id instead of static value
	ctx = context.WithValue(ctx, domain.Key("id"), int64(1))

	shortenedURL, err := h.Srv.CreateShortened(ctx, req.Original)
	var uErr *domain.UniqueError
	if err != nil && errors.As(err, &uErr) {
		return &api.CreateShortenedReply{Shortened: addr + shortenedURL},
			status.Errorf(codes.AlreadyExists, "provided URL already exist in the service")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "something went wrong in the service")
	}

	return &api.CreateShortenedReply{Shortened: addr + shortenedURL}, nil
}

func (h *Server) CreateShortenedFromBatch(ctx context.Context, req *api.CreateShortenedFromBatchRequest) (*api.CreateShortenedFromBatchReply, error) {
	if len(req.Original) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "got empty batch in request")
	}

	batch := make([]*domain.BatchElement, len(req.Original))
	for i, elem := range req.Original {
		batch[i] = &domain.BatchElement{ID: elem.Correlation, OriginalURL: elem.Original}
	}

	util.GetLogger().Infoln(batch)
	// TODO set actual user id
	ctx = context.WithValue(ctx, domain.Key("id"), int64(1))
	shortened, err := h.Srv.CreateShortenedFromBatch(ctx, batch, h.Wg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "something went wrong while processing the request")
	}

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	shortenedResult := &api.CreateShortenedFromBatchReply{}
	shortenedResult.Shortened = make([]*api.ShortenedWithCorrelation, len(shortened))
	for i, shrt := range shortened {
		shortenedResult.Shortened[i] = &api.ShortenedWithCorrelation{Correlation: shrt.ID, Shortened: addr + shrt.ShortenedURL}
	}

	return shortenedResult, nil
}

func (h *Server) ReadUserURLs(ctx context.Context, req *api.Empty) (*api.ReadUserURLsReply, error) {
	// TODO set actual user id
	ctx = context.WithValue(ctx, domain.Key("id"), int64(1))

	UserURLs, err := h.Srv.ReadUserURLs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request might be incorrect")
	}

	// TODO check if not authorized

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	userURLsReply := &api.ReadUserURLsReply{OriginalWithShortened: make([]*api.OriginalWithShortened, len(UserURLs))}
	for i, url := range UserURLs {
		userURLsReply.OriginalWithShortened[i] = &api.OriginalWithShortened{Original: url.OriginalURL, Shortened: addr + url.ShortURL}
	}

	return userURLsReply, nil
}

func (h *Server) ReadAmountOfURLsAndUsers(ctx context.Context, req *api.Empty) (*api.ReadAmountOfURLsAndUsersReply, error) {
	readAmountReply := &api.ReadAmountOfURLsAndUsersReply{}

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

func (h *Server) DeleteUserURLs(ctx context.Context, req *api.DeleteUserURLsRequest) (*api.Empty, error) {
	if len(req.UrlsToDelete) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "got zero urls in request")
	}

	ctx = context.WithValue(ctx, domain.Key("id"), int64(1))
	shortURLWithID := make([]domain.URLWithID, len(req.UrlsToDelete))
	for i, url := range req.UrlsToDelete {
		shortURLWithID[i] = domain.URLWithID{ID: ctx.Value(domain.Key("id")).(int64), URL: url}
	}

	go func() {
		h.Srv.DeleteUserURLs(ctx, shortURLWithID, h.ShortURLsChan, h.Once, h.Wg)
	}()

	return &api.Empty{}, nil
}
