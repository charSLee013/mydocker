package driver

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// 创建一个overlay2的文件系统
func NewWorkSpace( programUrl, containerID string) {

	//DEBUG
	Sugar.Debugf("New work space in : %v",OverlayDir + containerID)

	if err := CreateReadOnlyLayer(programUrl, containerID); err != nil {
		Sugar.Errorf("create lowerdir %s error %v", containerID+"/lower", err)
	}

	if err := CreateWriteLayer(containerID); err != nil {
		Sugar.Errorf("create writer lay %s error %v", containerID, err)
	}

}

// 创建只读的lower层
// 复制一份可执行文件过来
// 以及执行文件的动态链接库
// TODO: 支持java等的语言
func CreateReadOnlyLayer(programUrl, containerID string) error {
	lowerdirUrl := OverlayDir + containerID + "/" + "lower"

	exist, err := PathExists(lowerdirUrl)
	if err != nil {
		Sugar.Infof("Fail to judege whether lowerdir %s exists, %v", lowerdirUrl, err)
		return nil
	}

	// 注意权限！这里创建的是只读
	if !exist {
		if err := os.MkdirAll(lowerdirUrl, 0622); err != nil {
			Sugar.Errorf("Mkdir %s error %v", lowerdirUrl, err)
			return err
		}
	}

	// 复制可执行文件和依赖
	if err := copyProgram(programUrl,lowerdirUrl);err != nil {
		Sugar.Errorf("copy run program error %v",err)
	}

	return nil
}

// 复制执行文件以及动态链接库
func copyProgram (programUrl ,dstDir string) error {
	realProgramUrl, err := exec.LookPath(programUrl)

	if err != nil {
		Sugar.Errorf("Run program %s is not exist", realProgramUrl)
		return err
	} else {
		Sugar.Debugf("Run program path is : %s",realProgramUrl)
	}

	// readlink 在获取 /bin/sh 会变成 dash
	if realProgramUrl == "dash" {
		realProgramUrl = "/bin/sh"
	}

	programFolder := dstDir + "/" + filepath.Dir(realProgramUrl)

	// 判断是否在 . 或 / 下
	if programFolder == "." || programFolder == "/" {
		programFolder = ""
	}

	dstProgramUrl := dstDir + realProgramUrl
	if err := os.MkdirAll(programFolder, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", realProgramUrl, err)
		return err
	}

	// 把执行文件复制进去
	if err := copyFile(realProgramUrl, dstProgramUrl); err != nil {
		Sugar.Errorf("copy run programurl %s error %v", realProgramUrl, err)
	}
	
	// 再把执行文件所需要的动态链接库复制进去
	if err := copyDynamicLib(programUrl,dstDir);err != nil {
		Sugar.Errorf("copy dynamic lib error %v",err)
	}

	return nil
}

// 复制动态链接库到指定文件夹
func copyDynamicLib(programUrl ,destDir string) error {
	dynamicLib := findDynamicLib(programUrl)
	
	for _, libUrl := range dynamicLib{
		if err := copyFile(libUrl,destDir+libUrl);err != nil {
			return err
		}
	}
	return nil
}

func findDynamicLib(programUrl string) []string{
	cmd := exec.Command(programUrl)
	cmd.Env = []string{"LD_TRACE_LOADED_OBJECTS=1"}

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	stdoutMess,err := cmd.Output()
	stdoutMessStr := string(stdoutMess[:])
	if err != nil {
		log.Panic(err)
	}

	//切割字符串
	//得出依赖库的绝对路径
	dynamicLib := []string{}
	r, _ := regexp.Compile(`.*=> |\(0x.*|\t*`)
	messageLine := strings.Split(stdoutMessStr,"\n")

	for _,line := range messageLine[1:len(messageLine)-1]{
		line = r.ReplaceAllString(line,"")

		// clean space
		line = strings.Replace(line, " ", "", -1)

		dynamicLib = append(dynamicLib,line)
	}
	return dynamicLib
}

//拷贝文件  要拷贝的文件路径 拷贝到哪里
func copyFile(source, dest string) error {
	if source == "" || dest == "" {
		Sugar.Info("source or dest is null")
		return fmt.Errorf("src [%s] or dst [%s] is null")
	}

	// 查找文件真实路径
	realSourcePath,err := filepath.EvalSymlinks(source)
	if err != nil {
		Sugar.Errorf("find real link [%s] error %v",source,err)
		return err
	}

	// 判断文件是否存在
	if _,err := os.Stat(realSourcePath);os.IsNotExist(err) {
		Sugar.Errorf("copy file [%s] is not exist",realSourcePath)
		return err
	} else {
		Sugar.Debugf("copy file [%s] to %v",realSourcePath,dest)
	}

	//打开文件资源
	source_open, err := os.Open(realSourcePath)
	
	//养成好习惯。操作文件时候记得添加 defer 关闭文件资源代码
	if err != nil {
		return err
	}
	defer source_open.Close()

	fileInfo,err := source_open.Stat()
	if err != nil {
		return err
	}

	mode := fileInfo.Mode()
	folder := filepath.Dir(dest)


	// 以只读模式创建文件夹
	// 如果是软链接则创建的是软链接所在的文件夹
	if err := os.MkdirAll(folder,0622);err != nil {
		Sugar.Errorf("MkdirAll %s error %v",folder,err)
		return err
	}

	//只写模式打开文件 如果文件不存在进行创建 并赋予 744的权限。详情查看linux 权限解释
	dest_open, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	
	//养成好习惯。操作文件时候记得添加 defer 关闭文件资源代码
	defer dest_open.Close()
	
	//进行数据拷贝
	_, copy_err := io.Copy(dest_open, source_open)
	if copy_err != nil {
		return err
	} else {
		return nil
	}
}

// 创建并挂载 work,upper,merged
func CreateWriteLayer(containerID string) error {

	basedir := OverlayDir + containerID
	lowerdir := basedir + "/lower"

	workdir := basedir + "/work"
	if err := os.MkdirAll(workdir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
		return err
	}

	upperdir := basedir + "/upper"
	if err := os.MkdirAll(upperdir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
		return err
	}

	mergeddir := basedir + "/merged"
	if err := os.MkdirAll(mergeddir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
		return err
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, upperdir, workdir)

	// MS_NOSUID 文件系统执行程序时，不要使用该用户ID和组ID
	if err := syscall.Mount("none", mergeddir, "overlay", syscall.MS_NOSUID, opts); err != nil {
		Sugar.Errorf("mount overlay opts : %s error %v", opts, err)
		return err
	}

	return nil
}

// 删除容器overlay filesystem(仅保留lower层
func DeleteWorkSpace(containerID string) {
	if err := UnmountOverlay(containerID); err == nil {
		// umount 成功后才能删除其他overlay层
		DeleteWriteLayer(containerID)
	}
}

func UnmountOverlay(overlayDir string) error {
	mergeddir := OverlayDir + overlayDir + "/merged"

	if err := syscall.Unmount(mergeddir, 0); err != nil {
		Sugar.Errorf("Unmount %s error %v", overlayDir, err)
		return err
	}

	if err := os.RemoveAll(mergeddir); err != nil {
		Sugar.Errorf("Remove mountpoint dir %s error %v", mergeddir, err)
		return err
	}
	return nil
}

func DeleteVolumes(volumeURLs []string, layName string) error {
	mntURL := fmt.Sprintf(OverlayDir, layName)
	volume := mntURL + "/" + volumeURLs[1]

	if err := syscall.Unmount(volume, 0); err != nil {
		Sugar.Errorf("Umount volume %s error %v", volume, err)
		return err
	}

	return nil
}

// 从磁盘上删除overlay层
// 一定要先unmount后才进行删除！！！
func DeleteWriteLayer(containerID string) {
	workdir := OverlayDir + containerID + "/work"
	if err := os.RemoveAll(workdir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}

	upperdir := OverlayDir + containerID + "/upper"
	if err := os.RemoveAll(upperdir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}

	mergeddir := OverlayDir + containerID + "/merged"
	if err := os.RemoveAll(mergeddir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}
}


// 判断文件/文件夹是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
