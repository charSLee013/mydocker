package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"os"
)

var Sugar *zap.SugaredLogger

func init() {
	logger, err := InitLog()
	Sugar = logger.Sugar()
	if err != nil {
		panic(fmt.Sprintf("log 初始化失败: %v", err))
		os.Exit(1)
	}
}

const usage = "docker-cli is a simple container runtime inmplementation."

func main() {
	app := cli.NewApp()
	app.Name = "docker-cli"
	app.Usage = usage

}
