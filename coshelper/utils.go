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
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Get MD5 of the file. The MD5 string is uppercase. If MD5 calculation failed, return ""
func GetFileMd5(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Warnf("Cannot read file '%s'", path)
		return ""
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warnf("Cannot close file '%s'", path)
		}
	}()
	r := bufio.NewReader(f)
	h := md5.New()

	if _, err := io.Copy(h, r); err != nil {
		log.Warnf("Calculate MD5 of '%s' failed", path)
		return ""
	}

	return fmt.Sprintf("%X", h.Sum(nil))
}

// Convert JSON-style string to http.Header. If parsing failed, return nil.
func ConvertStringToHeader(str string) *http.Header {
	// Change single quotes to double quotes.
	str = strings.ReplaceAll(str, `\'`, "\a")
	str = strings.ReplaceAll(str, `'`, `"`)
	str = strings.ReplaceAll(str, "\a", `'`)
	var headerMap map[string]interface{}
	if err := json.Unmarshal([]byte(str), &headerMap); err != nil {
		log.Warn("HTTP header parse error. Ignore -H flag.")
		return nil
	}
	header := http.Header{}
	for k, v := range headerMap {
		header.Set(k, fmt.Sprintf("%v", v))
	}
	return &header
}

// Show a confirmation prompt and get input from user.
// default answer must be one of "", "yes", "no", otherwise a panic occurs.
// if "yes", "y", "ye" entered, or default answer is "yes" and user just pressed Enter, return true.
// if "no", "n" entered, or default answer is "no" and user just pressed Enter, return false.
// if other words entered, or default answer is "" and user just pressed Enter, show a message and continue read input.
func Confirm(question string, defaultAnswer string) bool {
	valid := map[string]bool{
		"yes": true,
		"y":   true,
		"ye":  true,
		"no":  false,
		"n":   false,
	}
	var prompt string
	switch defaultAnswer {
	case "":
		prompt = "[y/n] "
	case "yes":
		prompt = "[Y/n] "
	case "no":
		prompt = "[y/N] "
	default:
		panic("invalid default answer: " + defaultAnswer)
	}
	for {
		fmt.Print(question + prompt)
		var choice string
		_, _ = fmt.Scanln(&choice)
		if defaultAnswer != "" && choice == "" {
			return valid[defaultAnswer]
		}
		value, ok := valid[choice]
		if ok {
			return value
		}
		fmt.Println("Please respond with 'yes' or 'no' (or 'y' or 'n').")
	}
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func IsDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func IsFile(path string) bool {
	return FileExists(path) && !IsDir(path)
}

func GetFileSize(path string) (int64, error) {
	f, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return f.Size(), nil
}

// rawTime: 2019-05-24T10:56:40Z
// return: 2019-05-24 18:56:40 (localtime string)
func ConvertTime(rawTimeString string) string {
	theTime, err := time.Parse("2006-01-02T15:04:05Z", rawTimeString)
	if err != nil {
		return ""
	}
	theTime = theTime.Local()
	return theTime.Format("2006-01-02 15:04:05")
}

func Humanize(size int64, human bool) string {
	if !human {
		return fmt.Sprintf("%d", size)
	}
	if size > 1024*1024*1024 {
		return fmt.Sprintf("%.1fG",
			math.Round(float64(size)/(1024*1024*1024)))
	} else if size > 1024*1024 {
		return fmt.Sprintf("%.1fM",
			math.Round(float64(size)/(1024*1024)))
	} else if size > 1024 {
		return fmt.Sprintf("%.1fK",
			math.Round(float64(size)/1024))
	} else {
		return fmt.Sprintf("%d", size)
	}
}
