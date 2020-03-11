package driver

import ("os"
"go.uber.org/zap"
	"os/exec"
)

const (
	DriverURL = "/var/lib/gocker/overlay/"
)

var Sugar *zap.SugaredLogger

func InitLog(sugar *zap.SugaredLogger){
	Sugar = sugar
}

// 创建一个overlay2的文件系统
func NewWorkSpace(volume , layerName , containerName string) {
	CreateReadOnlyLayer()
}


// 创建只读的lower层
func CreateReadOnlyLayer(imageName string) error {
	lowerdirUrl := DriverURL + "/" + imageName

	exist,err := PathExists(lowerdirUrl)
	if err != nil {
		Sugar.Infof("Fail to judege whether lowerdir %s exists, %v",lowerdirUrl,err)
		return nil
	}

	if !exist {
		if err := os.MkdirAll(lowerdirUrl,0622);err != nil {
			Sugar.Errorf("Mkdir %s error %v",lowerdirUrl,err)
			return err
		}

		if _,err := exec.Command()
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
