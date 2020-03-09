package main

import (
	"github.com/charSLee013/mydocker/cgroups"
	"github.com/charSLee013/mydocker/container"
	"github.com/charSLee013/mydocker/network"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"log"
	"os"
)

var Sugar *zap.SugaredLogger

const usage = "docker-cli is a simple container runtime inmplementation."

func main() {
	cgroups.Init(Sugar)
	container.Init(Sugar)
	network.Init(Sugar)

	app := cli.NewApp()
	app.Name = "docker-cli"
	app.Usage = usage

	app.Commands = []*cli.Command{
		&initCommand,
		&runCommand,
		&listCommand,
		&logCommand,
		&execCommand,
		&stopCommand,
		&removeCommand,
		&commitCommand,
		&networkCommand,
	}

	app.Before = func(context *cli.Context) error {
		// set logger
		logger,err := InitLog()
		if err != nil {
			return err
		}
		Sugar = logger.Sugar()
		return nil
	}

	if err := app.Run(os.Args);err != nil {
		log.Fatal(err)
	}
}
