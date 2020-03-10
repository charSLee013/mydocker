package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//Create a Overlay2 filesystem as container root workspace
func NewWorkSpace(volume, imageName, containerName string) {

	//DEBUG
	Sugar.Debugf("overlay2 volume: %v, imageName : %v, containerName : %v",volume,imageName,containerName)

	CreateReadOnlyLayer(imageName)

	CreateWriteLayer(containerName)

	CreateMountPoint(containerName, imageName)

	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(volumeURLs, containerName)
			Sugar.Infof("NewWorkSpace volume urls %q", volumeURLs)
		} else {
			Sugar.Infof("Volume parameter input is not corrent.")
		}
	}
}

func MountVolume(volumeURLs []string, containerName string) error {
	parentUrl := volumeURLs[0]
	if err := os.MkdirAll(parentUrl, 0777); err != nil {
		Sugar.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}
	containerUrl := volumeURLs[1]
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerVolumeURL := mntURL + "/" + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		Sugar.Infof("Mkdir container dir %s error , %v", containerVolumeURL, err)
	}
	dirs := "dirs=" + parentUrl
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL).CombinedOutput()
	if err != nil {
		Sugar.Errorf("Mount volume failed. %v", err)
		return err
	}
	return nil
}

//Decompression tar image
func CreateReadOnlyLayer(imageName string) error {
	unTarFolderUrl := RootUrl + "/" + imageName + "/"
	imageUrl := RootUrl + "/" + imageName + ".tar"
	exist, err := PathExists(unTarFolderUrl)
	if err != nil {
		Sugar.Infof("Fail to judge whether dir %s exists, %v", unTarFolderUrl, err)
		return err
	}

	if !exist {
		if err := os.MkdirAll(unTarFolderUrl, 0622); err != nil {
			Sugar.Errorf("Mkdir %s error %v", unTarFolderUrl, err)
			return err
		}

		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", unTarFolderUrl).CombinedOutput(); err != nil {
			Sugar.Errorf("Untar dir %s error %v", unTarFolderUrl, err)
			return err
		}
	}
	return nil
}

func CreateWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {
		Sugar.Warnf("Mkdir write layer dir %s error.%v", writeURL, err)
	} else {
		Sugar.Infof("Mkdir write layer dir %s sucessed", writeURL)
	}
}

func CreateMountPoint(containerName, imageName string) error {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		Sugar.Errorf("Mkdir mountpoint dir %s error,%v", mntUrl, err)
		return err
	} else {
		Sugar.Infof("Mkdir mountpoint dir %s sucessed", mntUrl)
	}

	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := RootUrl + "/" + imageName
	dirs := "dirs=" + tmpWriteLayer + ":" + tmpImageLocation

	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl).CombinedOutput()
	if err != nil {
		Sugar.Errorf("Run command for creating mount  point failed %v \t %v \t %v ", err, dirs, mntUrl)
		return err
	}
	return nil
}

//Delete the AUFS filesystem while container exit
func DeleteWorkSpace(volume, containerName string) {
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteVolume(volumeURLs, containerName)
		}
	}
	DeleteMountPoint(containerName)
	DeleteWriteLayer(containerName)
}

func DeleteMountPoint(containerName string) error {
	mntURL := fmt.Sprintf(MntUrl, containerName)
	_, err := exec.Command("umount", mntURL).CombinedOutput()
	if err != nil {
		Sugar.Errorf("Unmount %s error %v", mntURL, err)
		return err
	}
	if err := os.RemoveAll(mntURL); err != nil {
		Sugar.Errorf("Remove mountpoint dir %s error %v", mntURL, err)
		return err
	}
	return nil
}

func DeleteVolume(volumeURLs []string, containerName string) error {
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerUrl := mntURL + "/" + volumeURLs[1]
	if _, err := exec.Command("umount", containerUrl).CombinedOutput(); err != nil {
		Sugar.Errorf("Umount volume %s failed. %v", containerUrl, err)
		return err
	}
	return nil
}

func DeleteWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.RemoveAll(writeURL); err != nil {
		Sugar.Infof("Remove writeLayer dir %s error %v", writeURL, err)
	}
}

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
