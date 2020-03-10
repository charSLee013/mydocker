package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/cgroups/subsystems"
	"github.com/charSLee013/mydocker/container"
	"github.com/charSLee013/mydocker/network"
	"github.com/urfave/cli/v2"
	"os"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit ie: gocker run -ti [image] [command]`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		&cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		&cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		&cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		&cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		&cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		&cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
		&cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		&cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
		},
	},
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("Missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args().Slice() {
			cmdArray = append(cmdArray, arg)
		}

		//get image name
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]

		createTty := context.Bool("ti")
		detach := context.Bool("d")

		if createTty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}
		Sugar.Infof("createTty %v", createTty)
		containerName := context.String("name")
		volume := context.String("v")
		network := context.String("net")

		envSlice := context.StringSlice("e")
		portmapping := context.StringSlice("p")

		Run(createTty, cmdArray, resConf, containerName, volume, imageName, envSlice, network, portmapping)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user`s process in container",
	Action: func(context *cli.Context) error {
		Sugar.Infow("Init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "Sugars",
	Usage: "print Sugars of a container",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("Please input ")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		//This is for callback
		if os.Getenv(ENV_EXEC_PID) != "" {
			Sugar.Infof("pid callback pid %s", os.Getpid())
			return nil
		}

		if context.Args().Len() < 2 {
			return fmt.Errorf("Missing container name or command")
		}

		containerName := context.Args().Get(0)
		var commandArray []string
		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}
		ExecContainer(containerName, commandArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("Missing container name")
		}

		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(9)
		removeContainer(containerName)
		return nil
	},
}

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 2 {
			return fmt.Errorf("Missing container name and image name")
		}

		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		commitContainer(containerName, imageName)
		return nil
	},
}

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []*cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				&cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args().Slice()[0])
				if err != nil {
					return fmt.Errorf("create network error :%+v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.DeleteNetwork(context.Args().Slice()[0])
				if err != nil {
					return fmt.Errorf("remove network error : %+v", err)
				}
				return nil
			},
		},
	},
}
