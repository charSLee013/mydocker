package network

import "go.uber.org/zap"

var Sugar *zap.SugaredLogger

func InitLog(sugar *zap.SugaredLogger) (){
	Sugar = sugar
}