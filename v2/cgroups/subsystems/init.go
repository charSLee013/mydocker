package subsystems

import "go.uber.org/zap"

var Sugar *zap.SugaredLogger

func InitLog(logger *zap.SugaredLogger){
	Sugar = logger
}
