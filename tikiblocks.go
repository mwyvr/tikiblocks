package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mwyvr/tikiblocks/util"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

var (
	blocks    []string
	channels  []chan bool
	signalMap map[string]int = make(map[string]int)
	outputTo  string         = "" // stdout, xsetroot, somebar
)

func init() {
	flag.StringVar(&outputTo, "o", outputTo, "output to: stdout, xroot, somebar")
}

func main() {
	// setup bar format and output method
	config := util.ReadConfig("tikiblocks.json")
	flag.Parse()
	outputTo := strings.ToLower(outputTo)
	if config.BarType == "" { // default to stdout
		config.BarType = "stdout"
	}
	if outputTo != "" {
		config.BarType = outputTo
	}
	config.OutputFile = util.SetOutput(config.BarType)

	channels = make([]chan bool, len(config.Actions))
	// recChannel is common for gothreads contributing to status bar
	recChannel := make(chan util.Change)
	for i, action := range config.Actions {
		// Assign a cell for each separator/prefix/action/suffix
		if config.Separator != "" {
			blocks = append(blocks, config.Separator)
		}
		if value, ok := action["prefix"]; ok {
			blocks = append(blocks, value.(string))
		}
		blocks = append(blocks, "action")
		actionId := len(blocks) - 1
		if value, ok := action["suffix"]; ok {
			blocks = append(blocks, value.(string))
		}
		// Create an unique channel for each action
		channels[i] = make(chan bool)
		signalMap["signal "+action["updateSignal"].(string)] = i
		if (action["command"].(string))[0] == '#' {
			go util.FunctionMap[action["command"].(string)](actionId, recChannel, channels[i], action)
		} else {
			go util.RunCmd(actionId, recChannel, channels[i], action)
		}
		timer := action["timer"].(string)
		if timer != "0" {
			go util.Schedule(channels[i], timer)
		}
	}
	go handleSignals(util.GetSIGRTchannel())
	oldstatus := ""
	// start event loop
	for {
		// Block until some gothread has an update
		res := <-recChannel
		if res.Success {
			blocks[res.BlockId] = res.Data
		} else {
			log.Println(res.Data)
			blocks[res.BlockId] = "ERROR"
		}
		oldstatus = updateBar(config, oldstatus)
	}
}

// Goroutine that pings a channel according to received signal
func handleSignals(rec chan os.Signal) {
	for {
		sig := <-rec
		if index, ok := signalMap[sig.String()]; ok {
			channels[index] <- true
		}
	}
}

// Craft status text out of blocks data
func updateBar(cfg util.Config, oldstatus string) string {
	var builder strings.Builder
	var status string
	if cfg.BarType == "somebar" {
		builder.WriteString("status")
	}
	for _, s := range blocks {
		builder.WriteString(s)
	}
	builder.WriteString("\n")
	status = builder.String()
	if oldstatus != status {
		fmt.Fprint(cfg.OutputFile, status)
	}
	return status
}

type xRootWriter interface {
	Write(p []byte) (n int, err error)
}

type xRoot struct{}

func newXRoot() {
	x, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	defer x.Close()
	root := xproto.Setup(x).DefaultScreen(x).Root
	_ = root
}
