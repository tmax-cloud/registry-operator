package operatorlog

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/robfig/cron"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	logFilePath = "/logs/operator.log"
)

var logger = ctrl.Log.WithName("operatorlog")

func LogFile() (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(logFilePath), os.ModePerm); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return file, nil
}

func StartDailyBackup(logFile *os.File) {
	// Logging Cron Job
	cronJob := cron.New()

	// Logging every day
	if err := cronJob.AddFunc("1 0 0 * * ?", func() {
		input, err := ioutil.ReadFile(logFilePath)
		if err != nil {
			fmt.Println(err)
			return
		}

		// backup yesterday log
		if err := ioutil.WriteFile(fmt.Sprintf("%s.%s.log", "operator", time.Now().AddDate(0, 0, -1).Format("2006-01-02")), input, 0644); err != nil {
			fmt.Println("failed to create log file", logFilePath)
			fmt.Println(err)
			return
		}

		logger.Info("backup log file successfully")

		// clear log file
		if err := os.Truncate(logFilePath, 0); err != nil {
			fmt.Println(err)
			return
		}

		if logFile == nil {
			fmt.Printf("%s log file is nil.\n", logFilePath)
			return
		}

		if _, err := logFile.Seek(0, io.SeekStart); err != nil {
			fmt.Println(err)
			return
		}
	}); err != nil {
		fmt.Println(err)
		return
	}

	logger.Info("cron job start for backup", "file path", logFilePath, "usage", fmt.Sprintf("mount %s directory", path.Dir(logFilePath)))
	cronJob.Start()
}
