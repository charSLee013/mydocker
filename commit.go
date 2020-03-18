package main

import (
	"archive/tar"
	"fmt"
	"github.com/charSLee013/mydocker/driver"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func commitContainer(layerName, imageName string) {
	layerDir := driver.OverlayDir + layerName

	// 如果merged层存在,则打包merged层
	// 如果merged层不存在,则打包lower层
	var srcTar string
	mergeddir := layerDir + "/merged"
	lowerdir := layerDir + "/lower"
	merged_stat, merged_err := os.Stat(mergeddir)
	if merged_err == nil && merged_stat.IsDir() != true {
		lower_stat, lower_err := os.Stat(lowerdir)
		if lower_stat.IsDir() != true && lower_err == nil {
			// 两个文件夹都不存在，则抛出错误
			Sugar.Errorf("The lower and merger folders in the [%s] do not exist", layerDir)
			return
		} else {
			srcTar = layerDir + "/lower"
		}
	} else {
		srcTar = layerDir + "/merged"
	}

	// 默认在当前目录下，或者在/tmp下面
	// TODO 支持重定向
	tarDst, merged_err := os.Getwd()
	if merged_err != nil {
		tarDst = "/tmp"
	}
	imageTar := tarDst + "/" + imageName + ".tar"

	if err := CreateTar(srcTar, imageTar); err != nil {
		Sugar.Errorf("Tar folder %s error %v", layerDir, err)
	}
}

// 将文件夹打包成 tar.gz
func CreateTar(src, dst string) error {
	//创建 tar文件
	tarfile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarwriter := tar.NewWriter(tarfile)
	// 如果关闭失败会造成tar包不完整
	defer func() {
		if err := tarwriter.Close(); err != nil {
			Sugar.Errorf("tar %s close error %v", src, err)
		}
	}()

	sfileInfo, err := os.Stat(src)
	if err != nil {
		Sugar.Errorf("%s is not exits", src)
		return err
	}

	// 判断打包对象是文件还是文件夹
	if !sfileInfo.IsDir() {
		return tarFile(dst, src, sfileInfo, tarwriter)
	} else {
		return tarFolder(src, tarwriter)
	}
}

func tarFile(directory string, filesource string, sfileInfo os.FileInfo, tarwriter *tar.Writer) error {
	sfile, err := os.Open(filesource)
	if err != nil {
		return err
	}
	defer sfile.Close()
	header, err := tar.FileInfoHeader(sfileInfo, "")
	if err != nil {
		return err
	}
	header.Name = directory
	err = tarwriter.WriteHeader(header)
	if err != nil {
		return err
	}

	if _, err = io.Copy(tarwriter, sfile); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func tarFolder(directory string, tarwriter *tar.Writer) error {
	var baseFolder string = filepath.Base(directory)

	// 便利文件夹下所有文件并加入到压缩包中
	return filepath.Walk(directory, func(targetpath string, file os.FileInfo, err error) error {

		if file == nil {
			Sugar.Warnf("file %s is not exits", file.Name())
			return err
		}

		if file.IsDir() {
			// information of file or folder
			header, err := tar.FileInfoHeader(file, "")
			if err != nil {
				return err
			}

			header.Name = filepath.Join(baseFolder, strings.TrimPrefix(targetpath, directory))
			fmt.Println(header.Name)
			if err = tarwriter.WriteHeader(header); err != nil {
				return err
			}

			os.Mkdir(strings.TrimPrefix(baseFolder, file.Name()), os.ModeDir)
			return nil
		} else {
			//baseFolder is the tar file path
			var fileFolder = filepath.Join(baseFolder, strings.TrimPrefix(targetpath, directory))
			return tarFile(fileFolder, targetpath, file, tarwriter)
		}
	})
}
