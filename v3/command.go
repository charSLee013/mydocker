package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/v2/cgroups/subsystems"
	"github.com/charSLee013/mydocker/v2/driver"
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