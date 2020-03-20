package main

import (
	"github.com/charSLee013/mydocker/v1/cgroups"
	"github.com/charSLee013/mydocker/v1/cgroups/subsystems"
	"github.com/charSLee013/mydocker/v1/driver"
	"math/rand"
	"os"
	"strings"
	"time"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, imageName string, envSlice []string) {

	// 生成containerID
	containerID := randStringBytes(16)

	// 创建隔离环境,并通过 pipe 的方式传递参数
	parent,writePipe := driver.NewParentProcess(tty, envSlice)
	if parent == nil {
		Sugar.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		Sugar.Error(err)
	}

	// cgroup name 使用containerID
	cgroupManager := cgroups.NewCgroupManager(containerID)
	defer cgroupManager.Destroy()

	// 创建并设置cgroup
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	// 传递参数给容器内实际运行的进程
	sendInitCommand(comArray,writePipe)

	if tty{
		// 等待前台进程完成
		parent.Wait()
	}
}


func randStringBytes(n int) string {
	letterBytes := "1234567890qwertyuioplkjhgfdsazxcvbnm"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}


func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	Sugar.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}