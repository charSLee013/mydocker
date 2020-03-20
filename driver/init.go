package driver

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

var Sugar *zap.SugaredLogger

func InitLog(sugar *zap.SugaredLogger) {
	Sugar = sugar
}

// 这里的init函数是在容器内部执行的,进到这步的时候容器已经创建出来了
// 使用 mount 去挂载 proc 文件系统
func RunContainerInitProcess() error {
	cmdArray := readUserCommand()

	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	setUpMount()

	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		Sugar.Errorf("Exec loop path error %v", err)
		return err
	}
	Sugar.Infof("Find path %s", path)

	// 容器起来的第一个进程也就是PID为1的进程
	// 是指定的前台进程
	// 实际上第一个进程是init初始化的进程
	// 如果不把第一个进程kill掉，那我们实际运行的进程才会变成前台进程（PID==1）
	// 但是 PID为1 的进程是不能被kill的，如果kill掉了容器也就退出了
	// 这里的 exec 调用就是黑魔法了
	// syscall.Exec 这个方法实际上是调用了 Kernel 的 init execveexecve(const char *filename, char *const argv[ ], char *const envp[ ])
	// execve()用来执行参数filename字符串所代表的文件路径，第二个参数是利用数组指针来传递给执行文件，并且需要以空指针(NULL)结束，最后一个参数则为传递给执行文件的新环境变量数组
	// 作用是将当前的进程替换成另一个进程，并且另一个进程会继承该进程的PID位,环境变量
	// 而该进程的代码段，堆栈都会被新进程给覆盖
	// 我们通过这种方法，将最初的init进程给覆盖掉
	// 这也是runC实现方式之一

	if err := syscall.Exec(path, cmdArray[0:], []string{}); err != nil {

		Sugar.Errorf(err.Error())

		//DEBUG
		Sugar.Debugf("path [%s]", path)

		for _, v := range os.Environ() {
			Sugar.Debugf("env : %v", v)
		}

		Sugar.Debugf("PID : %v", os.Getpid())

		f, err := os.Stat(path)
		if err != nil {
			Sugar.Error(err)
		} else {
			Sugar.Debug(f.Mode())
		}

		cmd := exec.Command(path)
		Sugar.Debug(cmd.Path)

		os.Exit(0)
	}

	return nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	defer pipe.Close()
	msg, err := ioutil.ReadAll(pipe)

	if err != nil {
		Sugar.Errorf("init read pipe error %v", err)
		return nil
	}

	msgStr := string(msg)

	// fix bug： 如果msgStr为空,Split 会返回一个[ ],len == 1 的数组
	// 判断字符串是否为空
	// 再进行切割
	if len(msgStr) > 0 {
		return strings.Split(msgStr, " ")
	} else {
		return []string{}
	}
}

/**
Init 挂载点
*/
func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		Sugar.Errorf("Get current location error %v", err)
		return
	}
	Sugar.Infof("Current location is %s", pwd)
	pivotRoot(pwd)

	//mount proc
	// MS_NOEXEC 不允许在挂上的文件系统上执行程序
	// MS_NOSUID 执行程序时，不遵照set-user-ID 和 set-group-ID位
	// MS_NODEV 不允许访问设备文件
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

// 将当前进程的root文件系统的移动到 new_root 成为新的root文件系统
func pivotRoot(root string) error {
	/**
	  为了使当前root的老 root 和新 root 不在同一个文件系统下，把root重新mount了一次
	  bind mount是把相同的内容换了一个挂载点的挂载方法
	*/

	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}

	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	// pivot_root 到新的rootfs, 现在老的 old_root 是挂载在rootfs/.pivot_root
	// 挂载点现在依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}

	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}

	// 删除临时文件夹
	return os.Remove(pivotDir)
}
