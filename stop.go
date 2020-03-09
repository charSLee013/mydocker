package main

import (
	"encoding/json"
	"fmt"
	"github.com/charSLee013/mydocker/container"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

func stopContainer(containerName string) {
	pid, err := GetContainerPidByName(containerName)
	if err != nil {
		Sugar.Errorf("Get container pid by name %s error %v")
		return
	}

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		Sugar.Errorf("Conver pid from string to int error %v", err)
		return
	}

	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		Sugar.Errorf("Stop container %s error %v", containerName, err)
		return
	}

	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		Sugar.Error("Get container %s into error %v", containerName, err)
		return
	}

	containerInfo.Status = container.STOP
	containerInfo.Pid = ""
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		Sugar.Errorf("Json marshal %s error %v", containerName, err)
		return
	}

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		Sugar.Errorf("Write file %s error ")
	}

}

func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		Sugar.Errorf("Read file %s error %v", configFilePath, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		Sugar.Errorf("GetContainerInfoByName unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

func removeContainer(containerName string) {
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		Sugar.Errorf("Get container %s info error %v", containerName, err)
		return
	}

	if containerInfo.Status != container.STOP {
		Sugar.Errorf("Couldn't remove running container")
		return
	}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		Sugar.Errorf("Remove file %s error %v", dirURL, err)
		return
	}
	container.DeleteWorkSpace(containerInfo.Volume, containerName)
}
