package MyLog

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type logInfo struct {
	msg    []byte
	length int64
}

type LOGLEVEL int

const (
	filePerm         os.FileMode = 0666
	LogLevel_DEBUG   LOGLEVEL    = 1
	LogLevel_WARNING LOGLEVEL    = 2
	LogLevel_INFO    LOGLEVEL    = 4
	LogLevel_ERROR   LOGLEVEL    = 8
	LogLevel_ALL     LOGLEVEL    = 15
)

var LogFile = `out` // log file name
var LogPath = ``    // log file sotre path
var LogExt = `.log` // log file ext

var LogMaxSize int64 = 1024 * 5 * 1024 // default single log file size (MB)
var LogBackupLimit = 30                // default log keep days
var LogLevel LOGLEVEL = LogLevel_ALL
var NoFileOutput = false // print log info to console only

var messageChan = make(chan logInfo, 1024)

func init() {
	fmt.Println(`... log system init`)
	go goWriter()
}

func getTomorrowTimestamp() time.Time {
	tom, _ := time.ParseInLocation("2006-01-02", time.Now().Local().AddDate(0, 0, 1).Format("2006-01-02"), time.Local)
	return tom
}

func goWriter() {
	cleanTimer := time.NewTimer(time.Until(getTomorrowTimestamp()))
	lastCleanFinished := true
	var logFileOpenedFlag = false

	var fileHandler *os.File = nil
	var fileLength int64 = 0
	var err error
	for {
		select {
		case <-cleanTimer.C:
			if fileHandler != nil {
				if !strings.Contains(fileHandler.Name(), `_`+time.Now().Local().Format(`20060102`)) {
					err = fileHandler.Close()
					if err != nil {
						fmt.Println(`error to close exist log file: ` + err.Error())
					}
					logFileOpenedFlag = false
				}
			}
			if lastCleanFinished {
				lastCleanFinished = false
				go func() {
					cleanLogFile()
					lastCleanFinished = true
				}()
			}
			cleanTimer = time.NewTimer(time.Until(getTomorrowTimestamp()))
			//fmt.Println(time.Until(getTomorrowTimestamp()))
		case info := <-messageChan:
			if !logFileOpenedFlag || (LogMaxSize > 0 && LogMaxSize < fileLength+info.length) {
				if fileHandler != nil {
					err = fileHandler.Close()
					if err != nil {
						fmt.Println(`error to close exist log file: ` + err.Error())
					}
				}
				fileHandler, fileLength, err = openLogFile(LogPath, info.length)
				if err != nil {
					fmt.Println(`error to open new log file: ` + err.Error())
					logFileOpenedFlag = false
				} else {
					logFileOpenedFlag = true
				}
			}

			if fileHandler != nil {
				_, err = fileHandler.Write(info.msg)
				if err != nil {
					fmt.Println(`error to write log file: ` + err.Error())
				} else {
					fileLength += info.length
				}
			}
		}
	}

}

func cleanLogFile() {
	if LogBackupLimit == 0 {
		return
	}
	rd, err := ioutil.ReadDir(LogPath)
	if err != nil {
		fmt.Println(`err to open log path: ` + LogPath)
		return
	}
	for _, fi := range rd {
		if fi.IsDir() {
			// do nothing
		} else {
			fn := fi.Name()
			if !strings.Contains(fn, LogFile) {
				continue
			}
			n := strings.Split(fn, LogFile)
			if len(n) >= 2 && len(n[1]) >= 9 {
				dateStr := n[len(n)-1][1:9] // log file name will be end with _dddddddd_xx.xxx or .xxx
				l, _ := time.LoadLocation("Local")
				d, err := time.ParseInLocation("20060102", dateStr, l)
				if err != nil {
					continue
				}
				s := int(time.Now().Sub(d).Hours() / 24)
				//fmt.Println(dateStr,s)
				if s >= LogBackupLimit { // >= if we set 30, the days between 10.01 and 10.31 is 30
					err = os.Remove(filepath.Join(LogPath, fi.Name()))
					if err != nil {
						fmt.Println(`error to delete old log: ` + err.Error())
					}
				}
			}
		}
	}

}

func logFileNameGenerator(date string, withFlag bool) string {
	if withFlag {
		stamp := time.Now().Format(`150405.000`)
		return LogFile + `_` + date + `_` + stamp[:6] + stamp[7:] + LogExt
	} else {
		return LogFile + `_` + date + LogExt
	}
}

func openLogFile(path string, dataLen int64) (*os.File, int64, error) {
	var fileLen int64
	var retFile *os.File
	retFile = nil

	LogPath = getLogPath()
	date := time.Now().Format("20060102")
	fileName := logFileNameGenerator(date, false)
	fullPath := filepath.Join(LogPath, fileName)

	fi, err := os.Stat(fullPath)
	if err != nil {
		fileLen = 0
	} else {
		fileLen = fi.Size()
	}

	if LogMaxSize > 0 && fileLen > 0 && LogMaxSize < dataLen+fileLen {
		err = os.Rename(fullPath, filepath.Join(LogPath, logFileNameGenerator(date, true)))
		if err != nil {
			fmt.Println(`error to rename log file: ` + err.Error())
		} else {
			fileLen = 0
		}
	}
	retFile, _ = os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, filePerm)

	return retFile, fileLen, nil
}

func getLogPath() string {
	if LogPath == `` {
		ostype := runtime.GOOS
		switch ostype {
		case `windows`:
			LogPath = `c:\log` //filepath.Dir(os.Args[0])
		default: // linux or other
			LogPath = `/var/log`
		}
	}

	if fi, err := os.Stat(LogPath); err != nil || !fi.IsDir() {
		err = os.MkdirAll(LogPath, filePerm)
		if err != nil {
			panic(`error to create log path: ` + err.Error())
		}
	}

	return LogPath
}

func Info(info string) {
	if LogLevel&LogLevel_INFO > 0 {
		writeLog(`[INFO]`, info)
	}
}

func Warning(info string) {
	if LogLevel&LogLevel_WARNING > 0 {
		writeLog(`[WARNING]`, info)
	}
}

func Error(info string) {
	if LogLevel&LogLevel_ERROR > 0 {
		writeLog(`[ERROR]`, info)
	}
}

func Debug(info string) {
	if LogLevel&LogLevel_DEBUG > 0 {
		writeLog(`[DEBUG]`, info)
	}
}

func writeLog(level string, info string) {
	strBuffer := bytes.Buffer{}

	strBuffer.WriteString(time.Now().Local().Format(`2006-01-02 15:04:05.000000 `))
	strBuffer.WriteString(level)
	strBuffer.WriteString(` `)
	strBuffer.WriteString(info)
	if info[len(info)-1] != '\n' {
		strBuffer.WriteByte('\n')
	}

	li := logInfo{
		msg:    strBuffer.Bytes(),
		length: int64(strBuffer.Len()),
	}

	if !NoFileOutput {
		messageChan <- li
	}
	_, err := os.Stdout.Write(strBuffer.Bytes())
	if err != nil {
		fmt.Println(`error to print log info: ` + err.Error())
	}
}
