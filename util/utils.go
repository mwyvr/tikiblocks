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

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Change struct {
	BlockId int
	Data    string
	Success bool
}

type Config struct {
	Separator  string
	BarType    string // stdout, stderr, xsetroot, somebar
	OutputFile io.Writer
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
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var byteValue []byte
	byteValue, err = io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal([]byte(byteValue), &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

type xRootWriter struct {
	Connection *xgb.Conn
	Root       xproto.Window
}

func (xw *xRootWriter) Write(barText []byte) (n int, err error) {
	length := len(barText)
	xproto.ChangeProperty(xw.Connection, xproto.PropModeReplace, xw.Root, xproto.AtomWmName, xproto.AtomString, 8, uint32(length), barText)
	return length, nil
}

func newXRootWriter() *xRootWriter {
	x, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	root := xproto.Setup(x).DefaultScreen(x).Root

	return &xRootWriter{
		Connection: x,
		Root:       root,
	}
}

func newSomebarWriter() io.Writer {
	var somebar *os.File
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		log.Fatal("XDG_RUNTIME_DIR not defined. dbus running?")
	}
	outputFn := path.Join(runtimeDir, "somebar-0")
	for i := 0; i < 100; i++ {
		somebar, err := os.OpenFile(outputFn, os.O_APPEND|os.O_WRONLY, 0x777)
		if err != nil {
			// somebar may not be up yet
			time.Sleep(10 * time.Millisecond)
		} else {
			return somebar
		}
	}
	if somebar == nil {
		log.Fatal("Unable to establish connection with somebar")
	}
	return somebar
}

func SetOutput(fname string) io.Writer {
	switch fname {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	case "xsetroot":
		return newXRootWriter()
	case "somebar":
		return newSomebarWriter()
	default:
		log.Fatal("Output must be one of stdout, stderr, xsetroot, somebar", fname)
	}
	return nil
}
