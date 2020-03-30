package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CpuLimitSubSystem struct {
}

func (s *CpuLimitSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {

		Sugar.Debugf("Set cpu.cfs_period_us : %v \t Set cpu.cfs_quota_us : %v",res.CpuPeriod,res.CpuQuota)

		if res.CpuPeriod != "" {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpu.cfs_period_us"), []byte(res.CpuPeriod), 0644); err != nil {
				return fmt.Errorf("set cgroup cpu share fail %v", err)
			}
		}

		if res.CpuQuota != "" {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath,"cpu.cfs_quota_us"), []byte(res.CpuQuota),0644);err != nil {
				return fmt.Errorf("set cgroup cpu.cfs_quota_us %s error %v",)
			}
		}
		return nil
	} else {
		return err
	}
}

func (s *CpuLimitSubSystem) Remove(cgroupPath string) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		return os.RemoveAll(subsysCgroupPath)
	} else {
		return err
	}
}

func (s *CpuLimitSubSystem) Apply(cgroupPath string, pid int) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {

		Sugar.Debugf("Add pid [%v] to %s",pid,s.Name())

		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *CpuLimitSubSystem) Name() string {
	return "cpu"
}
