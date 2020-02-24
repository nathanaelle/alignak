package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	sd "github.com/nathanaelle/sdialog/v2"
)

func validFile(file string) error {
	st, err := os.Stat(file)
	if err != nil {
		return err
	}

	mode := st.Mode()
	if !mode.IsRegular() || (mode.Perm()&0111) == 0 {
		return fmt.Errorf("%s : no exec permission", file)
	}
	return nil
}

func execCmd(ctx context.Context, cmd *exec.Cmd, reload <-chan struct{}, oups chan<- struct{}, wg *sync.WaitGroup) {
	defer cmd.Stdout.(io.WriteCloser).Close()
	defer cmd.Stderr.(io.WriteCloser).Close()

	wg.Add(1)
	defer wg.Done()

	if err := cmd.Start(); err != nil {
		sd.LogERR.LogError(err)
		oups <- struct{}{}
		return
	}

	sd.Notify(sd.Ready())

	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()

	exiting := false
	select {
	case <-reload:
		stopCmd(cmd)
		sd.Notify(sd.Reloading())

	case <-ctx.Done():
		exiting = true
		stopCmd(cmd)
		sd.Notify(sd.Stopping())

	case <-exited:
		if !exiting {
			if !cmd.ProcessState.Exited() {
				sd.LogALERT.Log("exit signal from an non terminated process")
			}
			sd.Notify(sd.Stopping())
			oups <- struct{}{}
		}
	}
}

func waitKill(cmd *exec.Cmd, sig syscall.Signal) <-chan struct{} {
	c := make(chan struct{})
	go func() {
		cmd.Process.Signal(sig)
		cmd.Process.Wait()
		close(c)
	}()
	return c
}

func tryKill(cmd *exec.Cmd, sig syscall.Signal) bool {
	sd.LogDEBUG.Logf("%s : send %s", cmd.Path, sig.String())

	select {
	case <-waitKill(cmd, sig):
	case <-time.After(500 * time.Millisecond):
	}

	return cmd.ProcessState.Exited()
}

func stopCmd(cmd *exec.Cmd) {
	if tryKill(cmd, syscall.SIGTERM) {
		return
	}

	if tryKill(cmd, syscall.SIGINT) {
		return
	}

	if tryKill(cmd, syscall.SIGKILL) {
		return
	}

	tryKill(cmd, syscall.SIGABRT)
}

func signalCatcher(wg *sync.WaitGroup) (context.Context, <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	reload := make(chan struct{})

	go func() {
		wg.Add(1)
		defer wg.Done()

		signalChannel := make(chan os.Signal)

		signal.Notify(signalChannel, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		defer close(signalChannel)
		defer close(reload)
		defer cancel()

		for sig := range signalChannel {
			switch sig {
			case syscall.SIGHUP:
				reload <- struct{}{}
			case syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM:
				return
			}
		}
	}()

	return ctx, reload
}
