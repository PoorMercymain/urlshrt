package util

import "go.uber.org/zap"

var (
	instance *zap.SugaredLogger
)

func InitLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	instance = logger.Sugar()
	return nil
}

func GetLogger() *zap.SugaredLogger {
	return instance
}
