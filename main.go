package main

import (
	"fmt"
	argParser "github.com/alexflint/go-arg"
	"github.com/google/logger"
	"github.com/sevlyar/go-daemon"
	"github.com/skiloop/servicemon/monitor"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type args struct {
	Verbose          bool          `arg:"-v" help:"verbose"`
	Foreground       bool          `arg:"-f" help:"work in foreground"`
	Restart          bool          `arg:"-r" help:"restart after instance exit"`
	RestartDelay     time.Duration `arg:"-d,--restart-delay" help:"restart delay, example: 1s"`
	Output           string        `arg:"-o" help:"output file"`
	Env              []string      `arg:"-e,separate" help:"additional env for service, multiple values, "`
	Checker          string        `arg:"-c" help:"script to check if process is healthy, if not healthy then program will stop"`
	Interval         time.Duration `arg:"-i" help:"checker interval, example: 1s"`
	Limit            uint64        `arg:"-l" help:"set open files limit"`
	Delay            time.Duration `arg:"-D" help:"checker delay after service start, example: 1s"`
	Result           string        `arg:"-R" help:"healthy checker result"`
	SecondaryCmd     string        `arg:"-s,--secondary-cmd" help:"secondary command, secondary will start when primary service is not healthy "`
	SecondaryOptions string        `arg:"-O,--secondary-options" help:"secondary options, if secondary command is not set then this is the secondary options for primary command"`
	PrimaryCmd       string        `arg:"positional,required,-m" help:"primary command"`
	Options          []string      `arg:"positional" help:"primary command options"`
}

func (args) Version() string {
	return "v0.1.0"
}

func main() {
	var a args
	argParser.MustParse(&a)
	if a.Foreground {
		//fmt.Fprintln(os.Stderr, "run in foreground")
		runService(&a)
	} else {
		//fmt.Fprintln(os.Stderr, "prepare to run in daemon")
		context := new(daemon.Context)
		child, err := context.Reborn()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "unable to run: %v\n", err)
		}
		if child != nil {
			// parent process go here
			return
		}
		defer context.Release()
		//fmt.Fprintln(os.Stderr, "run in daemon")
		runService(&a)
	}
}

func runService(args *args) {
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
			_, _ = fmt.Fprintf(os.Stderr, "Failed to open log file: %v", err)
		}
		if logFile == nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot open file: %s", args.Output)
			return
		}
		defer logFile.Close()
		defer logger.Init(name, false, false, logFile).Close()
	} else if args.Verbose {
		defer logger.Init(name, false, false, os.Stdout).Close()
	}

	logger.Info(name)
	if args.Limit > 0 {
		rlimit := syscall.Rlimit{}
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
		if err != nil {
			logger.Warningf("get rlimit error: %v", err)
		} else {
			logger.Infof("program number of open files: cur=%d, max=%d", rlimit.Cur, rlimit.Max)
			rlimit.Cur = args.Limit
			if rlimit.Cur > rlimit.Max {
				rlimit.Max = rlimit.Cur
			}
			_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
		}
		logger.Infof("open file limit %d", args.Limit)
	}

	start(args, logFile)
}

func start(a *args, logFile io.Writer) {
	var chk *monitor.Checker
	if a.Checker != "" {
		chk = &monitor.Checker{CMD: a.Checker, Options: "", OkResult: a.Result, Interval: a.Interval, Delay: a.Delay}
	} else {
		chk = nil
	}
	srv := &monitor.Service{Checker: chk, Program: a.PrimaryCmd, Options: a.Options, LogFile: logFile, AdditionalEnv: a.Env}
	var altSrv *monitor.Service
	if a.SecondaryCmd != "" || a.SecondaryOptions != "" {
		var cmd string
		if a.SecondaryCmd != "" {
			cmd = a.SecondaryCmd
		} else {
			cmd = a.PrimaryCmd
		}
		altSrv = &monitor.Service{Checker: chk, Program: cmd, Options: strings.Split(a.SecondaryOptions, " "), LogFile: logFile, AdditionalEnv: a.Env}
	}

	if a.Restart {
		runInRestartMode(altSrv, srv, a)
	} else {
		logger.Info("run service on single mode")
		err := srv.Run()
		if err != nil {
			logger.Infof("service run error: %v", err)
			if altSrv != nil {
				_ = altSrv.Run()
			}
		}
	}
}

func runInRestartMode(altSrv, srv *monitor.Service, a *args) {
	var curSrv *monitor.Service
	curSrv = srv
	logger.Info("run service on runInRestartMode mode")
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
		if a.Delay > 0 {
			time.Sleep(a.Delay)
		}
	}
}
