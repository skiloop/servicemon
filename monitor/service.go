package monitor

import (
	"io"
	"time"
	"github.com/google/logger"
	"os/exec"
	"os"
	"errors"
	"bytes"
	"strings"
)

type Checker struct {
	CMD      string
	Options  string
	OkResult string
	Interval time.Duration
	Delay    time.Duration
	running  bool
	cmdPath  string
	cmd      exec.Cmd
	buffer   bytes.Buffer
}

type Service struct {
	Program       string
	Options       []string
	AdditionalEnv []string
	pid           string
	Checker       *Checker
	LogFile       io.Writer
	cmd           *exec.Cmd
}

func (c *Checker) findChecker() error {
	cmdPath, err := exec.LookPath(c.CMD)
	if err == nil {
		c.cmdPath = cmdPath
	}
	return err
}

func (c *Checker) check() bool {
	cmd := exec.Command(c.cmdPath, strings.Split(c.Options, " ")...)
	cmd.Env = os.Environ()
	cmd.Stdout = &c.buffer
	cmd.Stdin = os.Stdin
	//cmd.Stderr = os.Stderr
	defer c.buffer.Reset()
	if err := cmd.Start(); err != nil {
		return false
	}
	if err := cmd.Wait(); err != nil {
		return false
	}
	res := ""
	buf := make([]byte, 1024)
	for {
		n, err := c.buffer.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return false
		}
		if n > 0 {
			res += string(buf[:n])
		}
	}
	logger.Infof("check result: %s/%s", res, c.OkResult)
	return c.OkResult == res
}

func (c *Checker) Run(callback func()) {
	c.running = true
	time.Sleep(c.Delay)

	for ; c.running; {
		if !c.check() {
			logger.Info("service check failed")
			c.running = false
			callback()
			break
		}
		time.Sleep(c.Interval)
	}
	logger.Info("checker ends")
}

func (srv *Service) copyEnv() {
	if srv.cmd != nil && srv.cmd.Env != nil {
		for i := range srv.AdditionalEnv {
			srv.cmd.Env = append(srv.cmd.Env, srv.AdditionalEnv[i])
		}
	} else {
		logger.Error("nil cmd or cmd Env")
	}
}
func (srv *Service) checkProgram() bool {
	logger.Info("check program")
	if srv.Checker != nil {
		err := srv.Checker.findChecker()
		if err != nil {
			logger.Errorf("check not found: %v", err)
			return false
		}
	}
	cmdPath, err := exec.LookPath(srv.Program)
	if err != nil {
		logger.Errorf("%s not found", srv.Program)
		return false
	}
	cmd := exec.Command(cmdPath, srv.Options...)
	cmd.Env = os.Environ()
	cmd.Stdout = srv.LogFile
	cmd.Stderr = srv.LogFile
	cmd.Stdin = os.Stdin

	srv.cmd = cmd
	srv.copyEnv()
	return true
}

func (srv *Service) Run() error {
	if !srv.checkProgram() {
		return errors.New("service check failed")
	}
	err := srv.start()
	if err != nil {
		return err
	}
	err = srv.cmd.Wait()
	if err != nil {
		logger.Error("wait error:", err)
	}
	return err
}
func (srv *Service) Stop() {
	if srv.cmd != nil {
		srv.cmd.Process.Kill()
		srv.cmd = nil
	}
}
func (srv *Service) start() error {
	logger.Info("start service")
	err := srv.cmd.Start()
	if err != nil {
		return err
	}
	if srv.Checker != nil {
		logger.Info("start checker")
		go srv.Checker.Run(srv.Stop)
	}
	return nil
}
