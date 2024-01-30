package util

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Change struct {
	BlockId int
	Data    string
	Success bool
}

type Config struct {
	Separator  string
	BarType    string // somebar, stdout; mandatory
	OutPutFile *os.File
	Actions    []map[string]interface{}
}

// Based on "timer" prorty from config file
// Schedule gothread that will ping other gothreads via send channel
func Schedule(send chan bool, duration string) {
	u, err := time.ParseDuration(duration)
	if err == nil {
		for range time.Tick(u) {
			send <- true
		}
	} else {
		log.Println("Couldn't set a scheduler due to improper time format: " + duration)
		send <- false
	}
}

// Run any arbitrary POSIX shell command
func RunCmd(blockId int, send chan Change, rec chan bool, action map[string]interface{}) {
	cmdStr := action["command"].(string)
	run := true

	for run {
		out, err := exec.Command("sh", "-c", cmdStr).Output()
		if err == nil {
			send <- Change{blockId, strings.TrimSuffix(string(out), "\n"), true}
		} else {
			send <- Change{blockId, err.Error(), false}
		}
		// Block until other thread will ping you
		run = <-rec
	}
}

// Create a channel that captures all 34-64 signals
func GetSIGRTchannel() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	sigArr := make([]os.Signal, 31)
	for i := range sigArr {
		sigArr[i] = syscall.Signal(i + 0x22)
	}
	signal.Notify(sigChan, sigArr...)
	return sigChan
}

// Read config and map it to configStruct
func ReadConfig(configName string) (config Config) {
	var confDir string
	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	var file *os.File
	file, err = os.Open(filepath.Join(confDir, "tikiblocks", configName))
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	var byteValue []byte
	byteValue, err = io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal([]byte(byteValue), &config)
	if err != nil {
		log.Fatal(err)
	}
	switch config.BarType {
	case "somebar":
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			log.Fatal("XDG_RUNTIME_DIR not defined. dbus running?")
		}
		outputFn := path.Join(runtimeDir, "somebar-0") // will fail on a multi-user system
		config.OutPutFile, err = os.OpenFile(outputFn, os.O_APPEND|os.O_WRONLY, 0x777)
		if err != nil {
			log.Fatal(err)
		}
	case "stdout":
		config.OutPutFile = os.Stdout
	default:
		log.Fatal("configuration file error: BarType must be defined as one of: somebar, stdout")
	}

	return config
}
