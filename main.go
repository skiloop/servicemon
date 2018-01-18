package main

import (
	"github.com/skiloop/servicemon/monitor"
	"os"
	"fmt"
	"time"
	"github.com/google/logger"
	argParser "github.com/alexflint/go-arg"
	"strings"
	"path/filepath"
	"github.com/sevlyar/go-daemon"
)

type Args struct {
	Verbose          bool     `arg:"-v" help:"verbose"`
	Foreground       bool     `arg:"-f" help:"work in foreground"`
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
	if args.Foreground {
		//fmt.Fprintln(os.Stderr, "run in foreground")
		runService(&args)
	} else {
		//fmt.Fprintln(os.Stderr, "prepare to run in daemon")
		context := new(daemon.Context)
		child, err := context.Reborn()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to run: %v\n", err)
		}
		if child != nil {
			// parent process go here
			return
		}
		defer context.Release()
		//fmt.Fprintln(os.Stderr, "run in daemon")
		runService(&args)
	}
}

func runService(args *Args) {
	var name string
	if len(os.Args) == 0 {
		name = "servicemon"
	} else {
		name = filepath.Base(os.Args[0])
	}
	// initial log
	//fmt.Println(args)
	logFile := os.Stdout

	var err error
	if args.Output != "" {
		//fmt.Println(args.Output)
		logFile, err = os.OpenFile(args.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v", err)
		}
		if logFile == nil {
			fmt.Fprintf(os.Stderr, "cannot open file: %s", args.Output)
			return
		}
		defer logFile.Close()
		defer logger.Init(name, false, false, logFile).Close()
	} else if args.Verbose {
		defer logger.Init(name, false, false, os.Stdout).Close()
	}

	logger.Info(name)

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
