package main

import (
	"github.com/Cooboob/MyLog"
)

func main() {
	MyLog.LogLevel = MyLog.LogLevel_ERROR | MyLog.LogLevel_WARNING
	MyLog.NoFileOutput = true
	MyLog.LogPath = `d:\test_log`
	MyLog.Debug(`debug`)
	MyLog.Info(`info`)
	MyLog.Warning(`warning`)
	MyLog.Error(`error`)
}
