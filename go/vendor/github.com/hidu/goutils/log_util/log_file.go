package log_util

import (
	"github.com/hidu/goutils/fs"
	"github.com/hidu/goutils/time_util"
	"log"
	"os"
	"time"
)

type LOG_TYPE string

const LOG_TYPE_DAY LOG_TYPE = "20060102"
const LOG_TYPE_HOUR LOG_TYPE = "2006010215"

func SetLogFile(loger *log.Logger, logPath string, log_type LOG_TYPE) error {
	var logFile *os.File
	var err error
	var checkFile = func() error {
		logPathCur := logPath + "." + time.Now().Format(string(log_type))
		if !fs.FileExists(logPathCur) {
			if logFile != nil {
				logFile.Close()
				logFile=nil
			}
			fs.DirCheck(logPathCur)
		}
		
		if logFile==nil{
			logFile, err = os.OpenFile(logPathCur, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				log.Println("create log file failed [", logPathCur, "]", err)
			}
			loger.SetOutput(logFile)
		}
		return err
	}
	checkFile()
	time_util.SetInterval(func() {
		checkFile()
	}, 1)
	return err
}
