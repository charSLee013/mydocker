package main

import (
	"encoding/json"
	"fmt"
	"github.com/charSLee013/mydocker/cgroups"
	"github.com/charSLee013/mydocker/cgroups/subsystems"
	"github.com/charSLee013/mydocker/driver"
	"github.com/charSLee013/mydocker/network"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, containerName, volume, imageName string,
	envSlice []string, nw string, portmapping []string) {
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}

	parent, writePipe := driver.NewParentProcess(tty, containerName, volume, imageName, envSlice)
	if parent == nil {
		Sugar.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		Sugar.Error(err)
	}

	//record container info
	containerName, err := recordContainerInfo(parent.Process.Pid, comArray, containerName, containerID, volume)
	if err != nil {
		Sugar.Errorf("Record container info error %v", err)
		return
	}

	// use containerID as cgroup name
	cgroupManager := cgroups.NewCgroupManager(containerID)
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	if nw != "" {
		// config container network
		network.Init()
		containerInfo := &driver.ContainerInfo{
			Id:          containerID,
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: portmapping,
		}
		if err := network.Connect(nw, containerInfo); err != nil {
			Sugar.Errorf("Error Connect Network %v", err)
			return
		}
	}

	// 传递参数给容器内实际运行进程
	sendInitCommand(comArray, writePipe)

	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
		driver.DeleteWorkSpace(volume, containerName)
	}

}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	Sugar.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

func recordContainerInfo(containerPID int, commandArray []string, containerName, id, volume string) (string, error) {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, "")
	containerInfo := &driver.ContainerInfo{
		Id:          id,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreatedTime: createTime,
		Status:      driver.RUNNING,
		Name:        containerName,
		Volume:      volume,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		Sugar.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)


	dirUrl := fmt.Sprintf(driver.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		Sugar.Errorf("Mkdir error %s error %v", dirUrl, err)
		return "", err
	}


	fileName := dirUrl + "/" + driver.ConfigName
	file, err := os.Create(fileName)
	defer file.Close()

	if err != nil {
		Sugar.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		Sugar.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
}

func deleteContainerInfo(containerId string) {
	dirURL := fmt.Sprintf(driver.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		Sugar.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
