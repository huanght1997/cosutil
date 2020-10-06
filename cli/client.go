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

package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gopkg.in/ini.v1"
)

type Client struct {
	Client *cos.Client
	Config *ClientConfig
}

type ClientConfig struct {
	SecretID     string
	SecretKey    string
	Token        string
	Bucket       string
	Endpoint     string
	MaxThread    int
	PartSize     int
	RetryTimes   int
	Timeout      int
	Schema       string
	VerifyMethod string
	Anonymous    bool
}

type PathPair struct {
	LocalPath string
	CosPath   string
}

const (
	VERSION = "1.8.6.21"
)

var (
	Region         string
	ConfigPath     string
	LogPath        string
	Bucket         string
	DebugMode      bool
	LogBackupCount int
	LogSize        int
)

// Create a new client.
func NewClient(config *ClientConfig) *Client {
	urlString := fmt.Sprintf("%s://%s.%s",
		config.Schema, config.Bucket, config.Endpoint)
	u, err := url.Parse(urlString)
	if err != nil {
		log.Fatalf("Failed to parse `%s`", urlString)
	}
	b := &cos.BaseURL{BucketURL: u}
	authTransport := cos.AuthorizationTransport{
		SecretID:  config.SecretID,
		SecretKey: config.SecretKey,
	}
	if config.Token != "" {
		authTransport.SessionToken = config.Token
	}
	client := cos.NewClient(b, &http.Client{
		Transport: &authTransport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	})
	return &Client{
		Client: client,
		Config: config,
	}
}

// Load config from config file path and return a ClientConfig.
func LoadConf(configPath string) *ClientConfig {
	fullConfigPath, err := homedir.Expand(configPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	cfg, err := ini.Load(fullConfigPath)
	if err != nil {
		log.Warnf("%s couldn't be found, please use 'cosutil config -h' to learn how to config cosutil!",
			fullConfigPath)
		log.Fatal(err.Error())
	} else {
		log.Debugf("%s is found", fullConfigPath)
	}

	var config ClientConfig
	section, err := cfg.GetSection("common")
	if section == nil || err != nil {
		log.Fatal("[common] section could not be found, please check your config file.")
	} else {
		if section.HasKey("secret_id") {
			config.SecretID = section.Key("secret_id").String()
		} else if section.HasKey("access_id") {
			config.SecretID = section.Key("access_id").String()
		}

		config.SecretKey = section.Key("secret_key").String()
		config.Token = section.Key("token").String()

		// Handle appid and bucket
		// ClientConfig has only one field `bucket`, but the input is various

		// Config file not specified and no argument parameter specified => quit now.
		if !section.HasKey("bucket") && Bucket == "" {
			log.Fatal("The configuration file is wrong. Check whether bucket has been specified")
		} else {
			// Read config file.
			// The bucket identifier of COS is <bucketName>-<AppId>.
			bucket := section.Key("bucket").String()
			if Bucket != "" {
				// If argument parameter specified, ignore config file.
				bucket = Bucket
			}
			if section.HasKey("appid") {
				appid := section.Key("appid").String()
				if strings.HasSuffix(bucket, "-"+appid) {
					// appid: appid, bucket: bucketname-appid
					config.Bucket = bucket
				} else {
					// appid: appid, bucket: bucketname
					config.Bucket = bucket + "-" + appid
				}
			} else {
				// no appid specified
				config.Bucket = bucket
			}
		}

		// Handle endpoint.
		if Region != "" {
			config.Endpoint = "cos." + compatible(Region) + ".myqcloud.com"
		} else if section.HasKey("region") {
			// If region specified, the endpoint is cos.<region>.myqcloud.com,
			// and the endpoint field in config file is ignored.
			region := compatible(section.Key("region").String())
			config.Endpoint = "cos." + region + ".myqcloud.com"
		} else if section.HasKey("endpoint") {
			config.Endpoint = section.Key("endpoint").String()
		}
		if config.Endpoint == "" {
			log.Fatal("The configuration file is wrong. Check whether region or endpoint has been specified")
		}

		config.MaxThread = getOrDefault(section, "max_thread", 5).(int)
		config.PartSize = getOrDefault(section, "part_size", 20).(int)
		config.RetryTimes = getOrDefault(section, "retry", 5).(int)
		config.Timeout = getOrDefault(section, "timeout", 60).(int)
		config.Schema = getOrDefault(section, "schema", "https").(string)
		config.VerifyMethod = getOrDefault(section, "verify", "md5").(string)

		anonymous := getOrDefault(section, "anonymous", "False").(string)
		config.Anonymous = strings.EqualFold(anonymous, "True")
	}
	log.Debugf("config parameter-> endpoint: %s, bucket: %s, part size: %d, max thread: %d",
		config.Endpoint, config.Bucket, config.PartSize, config.MaxThread)
	return &config
}

func updateProgress(progressbar *progressbar.ProgressBar, increment int64, finished chan bool) {
	_ = progressbar.Add64(increment)
	time.Sleep(100 * time.Millisecond)
	if progressbar.State().CurrentBytes >= float64(progressbar.GetMax64()) {
		finished <- true
	}
}

func getOrDefault(cfg *ini.Section, key string, defaultValue interface{}) interface{} {
	if cfg.HasKey(key) {
		switch defaultValue.(type) {
		case string:
			return cfg.Key(key).String()
		case int:
			return cfg.Key(key).MustInt(defaultValue.(int))
		case bool:
			return cfg.Key(key).MustBool(defaultValue.(bool))
		}
	}
	return defaultValue
}

func compatible(region string) string {
	if region == "" {
		return ""
	}
	if strings.HasPrefix(region, "cos.") {
		region = region[4:]
	}
	dict := map[string]string{
		"tj":       "ap-beijing-1",
		"bj":       "ap-beijing",
		"gz":       "ap-guangzhou",
		"sh":       "ap-shanghai",
		"cd":       "ap-chengdu",
		"spg":      "ap-singapore",
		"hk":       "ap-hongkong",
		"ca":       "na-toronto",
		"ger":      "eu-frankfurt",
		"cn-south": "ap-guangzhou",
		"cn-north": "ap-beijing-1",
	}
	result, ok := dict[region]
	if ok {
		return result
	}
	return region
}
