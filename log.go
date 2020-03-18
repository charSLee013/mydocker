package main

import (
	"fmt"
	"github.com/charSLee013/mydocker/driver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"os"
	"time"
)

func InitLog() (*zap.Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder, // 全路径编码器
	}

	// 设置日志级别
	atom := zap.NewAtomicLevelAt(zap.DebugLevel)

	config := zap.Config{
		Level:       atom, // 日志级别
		Development: true, // 开发模式，堆栈跟踪
		//Encoding:         "json",                                              // 输出格式 console 或 json
		Encoding:         "console",
		EncoderConfig:    encoderConfig,                                                            // 编码器配置
		InitialFields:    map[string]interface{}{"Date": time.Now().Format("2006-01-02 15:04:05")}, // 初始化字段，如：添加一个服务器名称
		OutputPaths:      []string{"stdout", "/var/log/mydocker.log"},                              // 输出到指定文件 stdout（标准输出，正常颜色） stderr（错误输出，红色）
		ErrorOutputPaths: []string{"stderr"},
	}

	// 构建日志
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	logger.Info("log 初始化成功")
	return logger, nil
}

func logContainer(containerName string) {
	dirURL := fmt.Sprintf(driver.DefaultInfoLocation, containerName)
	logFileLocation := dirURL + driver.ContainerLogFile

	file, err := os.Open(logFileLocation)
	defer file.Close()

	if err != nil {
		Sugar.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		Sugar.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}
	fmt.Fprint(os.Stdout, string(content))
}
