package main

import (
	"encoding/json"
	"fmt"
	"github.com/charSLee013/mydocker/driver"
	"io/ioutil"
	"os"
	"text/tabwriter"
)

func ListContainers() {
	dirURL := fmt.Sprintf(driver.DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL)-1]
	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		Sugar.Errorf("Read dir %s error %v", dirURL, err)
		return
	}

	var containers []*driver.ContainerInfo
	for _, file := range files {
		if file.Name() == "network" {
			continue
		}
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			Sugar.Errorf("Get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}

	if err := w.Flush(); err != nil {
		Sugar.Errorf("Flush error %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*driver.ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := fmt.Sprintf(driver.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + driver.ConfigName
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		Sugar.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo driver.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		Sugar.Errorf("Json unmarshal error %v", err)
		return nil, err
	}

	return &containerInfo, nil
}
