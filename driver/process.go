package driver

import (
	"os"
	"os/exec"
	"syscall"
)

const (
	RUNNING             string = "running"
	STOP                string = "stopped"
	EXIT                string = "exited"
	DefaultInfoLocation string = "/var/lib/gocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
	Dirver              string = "overlay"
	OverlayDir          string = "/var/lib/gocker/overlay/"
)

type ContainerInfo struct {
	Pid         string   `json:"pid"`         //容器的init进程在宿主机上的 PID
	Id          string   `json:"id"`          //容器Id
	Name        string   `json:"name"`        //容器名
	Command     string   `json:"command"`     //容器内init运行命令
	CreatedTime string   `json:"createTime"`  //创建时间
	Status      string   `json:"status"`      //容器的状态
	Volume      string   `json:"volume"`      //容器的数据卷
	PortMapping []string `json:"portmapping"` //端口映射
}

func NewParentProcess(tty bool, containerName, volume, imageName string, envSlice []string) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := NewPipe()
	if err != nil {
		Sugar.Errorf("New pipe error %v", err)
		return nil, nil
	}

	initCmd, err := os.Readlink("/proc/self/exe")
	if err != nil {
		Sugar.Errorf("get init process error %v", err)
		return nil, nil
	}

	cmd := exec.Command(initCmd, "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 将标准输出作为容器默认日志
		// 生产环境不推荐这么做
		// 最好用容器专用的ELK
		sugarLogPath := OverlayDir + containerName + "/" + containerName + ".log"

		// 文件不能关闭和删除
		// 如果想清空日志可以用  : > container.log 的方式
		stdSugarFile, err := os.Create(sugarLogPath)
		if err != nil {
			Sugar.Errorf("Create container stdout %s error %v", sugarLogPath, err)
			return nil, nil
		}
		cmd.Stdout = stdSugarFile
	}

	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(os.Environ(), envSlice...)

	NewWorkSpace(volume, imageName, containerName)

	// 容器默认挂载位置在 merged 层
	cmd.Dir = OverlayDir + containerName + "/merged"

	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
