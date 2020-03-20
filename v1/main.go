package main

import (
	"github.com/charSLee013/mydocker/v1/cgroups"
	"github.com/charSLee013/mydocker/v1/driver"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"log"
	"os"
)

var Sugar *zap.SugaredLogger

const usage = "docker-cli is a simple container runtime inmplementation."

func main() {
	app := cli.NewApp()
	app.Name = "docker-cli"
	app.Usage = usage

	app.Commands = []*cli.Command{
		&runCommand,
		&initCommand,
	}

	// set logger
	logger, err := InitLog()
	if err != nil {
		log.Fatal(err)
	}
	Sugar = logger.Sugar()
	cgroups.InitLog(Sugar)
	driver.InitLog(Sugar)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
