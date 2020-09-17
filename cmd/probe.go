package cmd

import (
	"crypto/rand"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"cosutil/cli"
	. "cosutil/coshelper"

	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	probeNum, probeSize int
	probeCmd            = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "probe [-h] [-n NUM] [-s SIZE]",
		Short:                 "Connection test",
		RunE:                  probe,
	}
)

func init() {
	rootCmd.AddCommand(probeCmd)

	probeCmd.Flags().SortFlags = false
	probeCmd.Flags().IntVarP(&probeNum, "num", "n", 3,
		"Specify test times")
	probeCmd.Flags().IntVarP(&probeSize, "size", "s", 1,
		"Specify test filesize(unit MB)")
}

func probe(*cobra.Command, []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)

	filename := "tmp_test_" + strconv.Itoa(probeSize) + "M"
	var timeUpload, timeDownload int64 = 0, 0
	var maxTimeUpload, maxTimeDownload int64 = 0, 0
	var minTimeUpload, minTimeDownload int64 = math.MaxInt64, math.MaxInt64
	successNum := 0

	err := genRandomFile(filename, probeSize)
	if err != nil {
		log.Warn("Create testfile failed")
		log.Info("[failure]")
		return Error{
			Code:    -1,
			Message: "probe fail",
		}
	}
	for i := 0; i < probeNum; i++ {
		header := &http.Header{}
		timeStart := time.Now().UnixNano()
		ret := client.UploadFile(filename, filename, header, &cli.UploadOption{
			SkipMd5: true,
			Sync:    false,
			Include: []string{"*"},
			Ignore:  []string{""},
			Force:   true,
		})
		timeEnd := time.Now().UnixNano()
		if maxTimeUpload < timeEnd-timeStart {
			maxTimeUpload = timeEnd - timeStart
		}
		if minTimeUpload > timeEnd-timeStart {
			minTimeUpload = timeEnd - timeStart
		}
		timeUpload += timeEnd - timeStart
		if ret != 0 {
			log.Info("[failure]")
			continue
		}
		log.Info("[success]")
		timeStart = time.Now().UnixNano()
		ret = client.DownloadFile(filename, filename, header, &cli.DownloadOption{
			Force:   true,
			Sync:    false,
			Num:     10,
			Ignore:  []string{""},
			Include: []string{"*"},
			SkipMd5: true,
		})
		timeEnd = time.Now().UnixNano()
		if maxTimeDownload < timeEnd-timeStart {
			maxTimeDownload = timeEnd - timeStart
		}
		if minTimeDownload > timeEnd-timeStart {
			minTimeDownload = timeEnd - timeStart
		}
		timeDownload += timeEnd - timeStart
		if ret != 0 {
			log.Info("[failure]")
			continue
		}
		log.Info("[success]")
		successNum++
	}
	log.Infof("Success Rate: [%d/%d]", successNum, probeNum)
	_ = os.Remove(filename)
	if successNum == probeNum {
		t := table.NewWriter()
		t.AppendHeader(table.Row{
			strconv.Itoa(probeSize) + "M TEST",
			"Average", "Min", "Max",
		})
		t.Style().Options.DrawBorder = false
		avgUploadBw := Humanize(int(float32(probeSize)*float32(successNum)*1024*1024/(float32(timeUpload)*1e-9)), true) + "B/s"
		avgDownloadBw := Humanize(int(float32(probeSize)*float32(successNum)*1024*1024/(float32(timeDownload)*1e-9)), true) + "B/s"
		minUploadBw := Humanize(int(float32(probeSize)*1024*1024/(float32(maxTimeUpload)*1e-9)), true) + "B/s"
		minDownloadBw := Humanize(int(float32(probeSize)*1024*1024/(float32(maxTimeDownload)*1e-9)), true) + "B/s"
		maxUploadBw := Humanize(int(float32(probeSize)*1024*1024/(float32(minTimeUpload)*1e-9)), true) + "B/s"
		maxDownloadBw := Humanize(int(float32(probeSize)*1024*1024/(float32(minTimeDownload)*1e-9)), true) + "B/s"
		t.AppendRow(table.Row{
			"Upload",
			avgUploadBw, minUploadBw, maxUploadBw,
		})
		t.AppendRow(table.Row{
			"Download",
			avgDownloadBw, minDownloadBw, maxDownloadBw,
		})
		log.Info("\n" + t.Render())
		return nil
	}
	return Error{
		Code:    -1,
		Message: "probe failed",
	}
}

func genRandomFile(filename string, fileSize int) error {
	f, err := os.Create(filename)
	if err != nil {
		log.Warn(err.Error())
		return err
	}
	defer func() {
		err = f.Close()
	}()
	buf := make([]byte, 1024)
	for i := 0; i < fileSize*1024; i++ {
		_, err = rand.Read(buf)
		if err != nil {
			log.Warn(err.Error())
			return err
		}
		_, err = f.Write(buf)
		if err != nil {
			log.Warn(err.Error())
			return err
		}
	}
	return err
}
