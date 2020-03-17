package driver

import (
	"fmt"
	"os"
	"syscall"
)

const (
	DriverURL = "/var/lib/gocker/overlay/"
)

// 创建一个overlay2的文件系统
func NewWorkSpace(volume, layerName, containerName string) {
	CreateReadOnlyLayer(layerName)
	CreateWriteLayer(layerName)

	if volume != "" {
		Sugar.Infof("now not support mount volume %s", volume)
	}
}

// 创建只读的lower层
func CreateReadOnlyLayer(layerName string) error {
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
	return nil
}

// 创建并挂载 work,upper,merged
func CreateWriteLayer(layerName string) {
	basedir := OverlayDir + layerName
	lowerdir := basedir + "/lower"

	workdir := basedir + "/work"
	if err := os.MkdirAll(workdir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
	}

	upperdir := basedir + "/upper"
	if err := os.MkdirAll(upperdir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
	}

	mergeddir := basedir + "/merged"
	if err := os.MkdirAll(mergeddir, 0755); err != nil {
		Sugar.Errorf("Mkdir %s error %v", workdir, err)
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, upperdir, workdir)

	// MS_NOSUID 文件系统执行程序时，不要使用该用户ID和组ID
	if err := syscall.Mount("none", mergeddir, "overlay", syscall.MS_NOSUID, opts); err != nil {
		Sugar.Errorf("mount overlay opts : %s error %v", opts, err)
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
