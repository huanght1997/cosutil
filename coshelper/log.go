/*
Copyright © 2020 Haitao Huang <hht970222@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package coshelper

import (
	"io"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/t-tomalak/logrus-easy-formatter"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(logFilePath string, logSize int, logBackupCount int, debugMode bool) {
	fullLogFilePath, err := homedir.Expand(logFilePath)
	if err != nil {
		logrus.SetOutput(os.Stderr)
		logrus.Warn("Cannot access " + logFilePath)
		return
	}
	logger := &lumberjack.Logger{
		Filename:   fullLogFilePath,
		MaxSize:    logSize,
		MaxBackups: logBackupCount,
	}
	if debugMode {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05 MST",
		LogFormat:       "%time% - [%lvl%]: %msg%\n",
	})
	logrus.SetOutput(io.MultiWriter(os.Stderr, logger))
}
