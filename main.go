package main

import (
	"github.com/skiloop/servicemon/monitor"
	"os"
	"fmt"
	"time"
	"github.com/google/logger"
	argParser "github.com/alexflint/go-arg"
	"strings"
)

type Args struct {
	Verbose          bool     `arg:"-v" help:"verbose"`
	Restart          bool     `arg:"-r" help:"restart after instance exit"`
	RestartDelay     int64    `arg:"-d,--restart-delay" help:"restart delay"`
	Output           string   `arg:"-o" help:"output file"`
	Env              []string `arg:"-e,separate" help:"env for service, can use more than once"`
	Checker          string   `arg:"-c" help:"script to check if process is healthy, if not healthy then program will stop"`
	Interval         int64    `arg:"-i" help:"checker interval"`
	Delay            int64    `arg:"-D" help:"checker delay after service start"`
	Result           string   `arg:"-R" help:"healthy checker result"`
	SecondaryCmd     string   `arg:"-s,--secondary-cmd" help:"secondary command, secondary will start when primary service is not healthy "`
	SecondaryOptions string   `arg:"-O,--secondary-options" help:"secondary options, if secondary command is not set then this is the secondary options for primary command"`
	PrimaryCmd       string   `arg:"positional,required,-m" help:"primary command"`
	Options          []string `arg:"positional" help:"primary command options"`
}

func (Args) Version() string {
	return "0.1.0"
}

func main() {
	var args Args
	argParser.MustParse(&args)
	// initial log
	fmt.Println(args)
	logFile := os.Stdout
	if args.Verbose {
		var err error
		if args.Output != "" {
			logFile, err = os.Open(args.Output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot open file: %s", args.Output)
				return
			}
			defer logFile.Close()
			defer logger.Init("main", true, false, logFile).Close()
		} else {
			defer logger.Init("main", false, false, os.Stdout).Close()
		}
	} else {
		logFile, _ = os.Open(os.DevNull)
		defer logger.Init("main", false, true, logFile).Close()
	}
	logger.Info("main")

	var chk *monitor.Checker
	if args.Checker != "" {
		chk = &monitor.Checker{CMD: args.Checker, Options: "", OkResult: args.Result, Interval: args.Interval, Delay: args.Delay}
	} else {
		chk = nil
	}

	srv := &monitor.Service{Checker: chk, Program: args.PrimaryCmd, Options: args.Options, LogFile: logFile}
	var altSrv *monitor.Service
	if args.SecondaryCmd != "" || args.SecondaryOptions != "" {
		var cmd string
		if args.SecondaryCmd != "" {
			cmd = args.SecondaryCmd
		} else {
			cmd = args.PrimaryCmd
		}
		altSrv = &monitor.Service{Checker: chk, Program: cmd, Options: strings.Split(args.SecondaryOptions, " "), LogFile: logFile}
	}
	var curSrv *monitor.Service
	curSrv = srv
	if args.Restart {
		logger.Info("run service on restart mode")
		for {
			err := curSrv.Run()
			if err != nil {
				logger.Infof("service run error: %v", err)
				if altSrv != nil {
					if curSrv == altSrv {
						curSrv = srv
					} else {
						curSrv = altSrv
					}
				}
			}
			if args.Delay > 0 {
				time.Sleep(time.Duration(args.Delay * time.Second.Nanoseconds()))
			}
		}
	} else {
		logger.Info("run service on single mode")
		err := srv.Run()
		if err != nil {
			logger.Infof("service run error: %v", err)
			if altSrv != nil {
				altSrv.Run()
			}
		}
	}

}
