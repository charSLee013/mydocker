package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/v1/cgroups/subsystems"
	"github.com/charSLee013/mydocker/v1/driver"
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
		programName := cmdArray[0]
		//cmdArray = cmdArray[1:]

		createTty := context.Bool("ti")
		detach := context.Bool("d")

		if createTty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuPeriod:    context.String("cpu-period"),
			CpuQuota:   context.String("cpu-quota"),
			CpuSet: context.String("cpuset"),
		}
		Sugar.Infof("createTty %v", createTty)

		envSlice := context.StringSlice("e")

		//DEBUG
		//Sugar.Debugf("enable tty : %v ",createTty)
		//Sugar.Debugf("cmdArray : %v",cmdArray)
		//Sugar.Debugf("resConf : %v",resConf)

		Run(createTty, cmdArray, resConf, programName, envSlice)
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