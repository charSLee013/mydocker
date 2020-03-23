package driver

import (
	"os"
	"os/exec"
	"path"
	"syscall"
)

const (
	DefaultInfoLocation string = "/var/lib/gocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
	Dirver              string = "overlay"
	OverlayDir          string = "/var/lib/gocker/overlay/"
)


func NewParentProcess(tty bool, containerID, programUrl string, envSlice []string) (*exec.Cmd, *os.File) {

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

	// 通过 cgroup 隔离环境
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	if tty{
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 将标准输出作为容器默认日志
		// 生产环境不推荐这么做
		// 最好用容器专用的ELK
		sugarLogPath := OverlayDir + containerID + "/" + containerID + ".log"

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
	cmd.Dir = path.Join(OverlayDir, containerID,"merged")

	//
	NewWorkSpace(programUrl, containerID)

	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
