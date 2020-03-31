package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/v3/cgroups/subsystems"
	"github.com/charSLee013/mydocker/v3/driver"
	"github.com/charSLee013/mydocker/v3/network"
	"github.com/urfave/cli/v2"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit ie: gocker run -ti [image] [command]`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name: "ti",
			Usage: "enable tty",
		},
		&cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		&cli.StringFlag{
			Name:  "cpu-period",
			Usage: "cpu.cfs_period_us limit",
		},
		&cli.StringFlag{
			Name:  "cpu-quota",
			Usage: "cpu.cfs_quota_us limit",
		},
		&cli.StringFlag{
			Name: "cpuset",
			Usage: "cpuset limit",
		},
		&cli.StringFlag{
			Name: "v",
			Usage: "volume",
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

		//get run program url
		programURL := cmdArray[0]

		createTty := context.Bool("ti")

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuPeriod:    context.String("cpu-period"),
			CpuQuota:   context.String("cpu-quota"),
			CpuSet: context.String("cpuset"),
		}
		Sugar.Infof("createTty %v", createTty)

		envSlice := context.StringSlice("e")

		//DEBUG
		Sugar.Debugf("tty : %v \t cmdArray : %v , programUrl : %v",createTty,cmdArray,programURL)

		Run(createTty, cmdArray , resConf, programURL, envSlice)
		return nil
	},
}

var initCommand = cli.Command{
	Name: "init",
	Usage: "Init container process run program in container",
	Action: func(context *cli.Context) error {
		Sugar.Info("Init come on")
		err := driver.RunContainerInitProcess()
		return err
	},
}

var networkCommand = cli.Command{
	Name: "network",
	Usage: "container network commands",
	Subcommands: []*cli.Command{
		{
			Name:"create",
			Usage:"create a container network",
			Flags:[]cli.Flag{
				&cli.StringFlag{
					Name:"driver",
					Usage:"network driver",
				},
				&cli.StringFlag{
					Name:"subnet",
					Usage:"subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.CreateNetwork(context.String("driver"),context.String("subnet"),context.Args().Slice()[0])
				if err != nil {
					return fmt.Errorf("create network error: %+v",err)
				}
				return nil
			},
		},
		{
			Name:"list",
			Usage:"list container network",
			Action: func(context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:"remove",
			Usage:"remove container network",
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("Missing network name")
				}

				network.Init()
				err := network.DeleteNetwork(context.Args().Slice()[0])
				if err != nil {
					return fmt.Errorf("remove network error: %+v",err)
				}
				return nil
			},
		},
	},
}