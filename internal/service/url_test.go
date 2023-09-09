package service

import (
	"context"
	"testing"

	"github.com/PoorMercymain/urlshrt/internal/domain/mocks"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestReadUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ur := mocks.NewMockURLRepository(ctrl)

	ur.EXPECT().ReadUserURLs(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()

	us := NewURL(ur)

	jsonStrSlice, err := us.ReadUserURLs(context.Background())
	require.NoError(t, err)
	require.Len(t, jsonStrSlice, 0)
}

func TestPingPg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ur := mocks.NewMockURLRepository(ctrl)

	ur.EXPECT().PingPg(gomock.Any()).Return(nil).AnyTimes()

	us := NewURL(ur)

	err := us.PingPg(context.Background())
	require.NoError(t, err)
}
