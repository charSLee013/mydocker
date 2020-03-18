package driver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	DriverURL = "/var/lib/gocker/overlay/"
)

// 创建一个overlay2的文件系统
func NewWorkSpace(volume, imageName, layerName string) {
	if err := CreateReadOnlyLayer(imageName, layerName); err != nil {
		Sugar.Errorf("create lowerdir %s error %v", layerName+"/lower", err)
	}

	if err := CreateWriteLayer(layerName); err != nil {
		Sugar.Errorf("create writer lay %s error %v", layerName, err)
	}

	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			//TODO
			Sugar.Infof("volume.len >=2 and volume is %v", volumeURLs)
		} else {
			Sugar.Infof("Volume parmeter input is not correct")
		}
	}
}

// 创建只读的lower层
// 复制一份可执行文件过来
// TODO: 支持java等的语言
func CreateReadOnlyLayer(imageName, layerName string) error {
	lowerdirUrl := DriverURL + "/" + layerName + "/" + "lower"

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

	// readlink 在获取 /bin/sh 会变成 dash
	//programUrl, err := os.Readlink(imageName)
	//if err != nil {
	//	Sugar.Errorf("Run program %s is not exist",programUrl)
	//	return err
	//}
	programUrl := imageName

	programPath := lowerdirUrl + "/" + filepath.Dir(programUrl)
	// 判断是否在 . 或 / 下
	if programPath == "." || programPath == "/" {
		programPath = ""
	}

	dstProgramUrl := lowerdirUrl + programUrl
	if err := os.MkdirAll(programPath, 0700); err != nil {
		Sugar.Errorf("Mkdir %s error %v", programUrl, err)
		return err
	}

	if err := copyFile(programUrl, dstProgramUrl); err != nil {
		Sugar.Errorf("copy run programurl %s error %v", programUrl, err)
	}

	return nil
}

//拷贝文件  要拷贝的文件路径 拷贝到哪里
func copyFile(source, dest string) error {
	if source == "" || dest == "" {
		Sugar.Info("source or dest is null")
		return fmt.Errorf("src--%s or dst--%s is null")
	}
	//打开文件资源
	source_open, err := os.Open(source)
	//养成好习惯。操作文件时候记得添加 defer 关闭文件资源代码
	if err != nil {
		return err
	}
	defer source_open.Close()
	//只写模式打开文件 如果文件不存在进行创建 并赋予 644的权限。详情查看linux 权限解释
	dest_open, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 644)
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
func CreateWriteLayer(layerName string) error {
	basedir := OverlayDir + layerName
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
func DeleteWorkSpace(volueme, layerName string) {
	if volueme != "" {
		volumeURLs := strings.Split(volueme, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteVolumes(volumeURLs, layerName)
		}
	}

	if err := DeleteMountPoint(layerName); err == nil {
		// umount 成功后才能删除其他overlay层
		DeleteWriteLayer(layerName)
	}
}

func DeleteMountPoint(overlayDir string) error {
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
func DeleteWriteLayer(layerName string) {
	workdir := OverlayDir + layerName + "/workdir"
	if err := os.RemoveAll(workdir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}

	upperdir := OverlayDir + layerName + "/upper"
	if err := os.RemoveAll(upperdir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}

	mergeddir := OverlayDir + layerName + "/merged"
	if err := os.RemoveAll(mergeddir); err != nil {
		Sugar.Warnf("Remove overlay %s error %v", workdir, err)
	}
}

//TODO
//func MountVolume(volumeURLs []string, containerName string) error {
//	parentUrl := volumeURLs[0]
//	if err := os.MkdirAll(parentUrl,0700);err != nil {
//
//	}
//}

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
