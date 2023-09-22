package util

import "go.uber.org/zap"

var instance *zap.SugaredLogger

// InitLogger is a function to initialize logger.
func InitLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	instance = logger.Sugar()
	return nil
}

// GetLogger is a function to get logger's pointer.
func GetLogger() *zap.SugaredLogger {
	return instance
}
