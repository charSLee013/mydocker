package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/container"
	"os/exec"
)

func commitContainer(containerName, imageName string) {
	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/"

	imageTar := container.RootUrl + "/" + imageName + ".tar"

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		Sugar.Errorf("Tar folder %s error %v", mntURL, err)
	}
}
