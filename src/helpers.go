package	main

import	(
	"io"
	"os"
	"fmt"
	"sync"
	"time"
	"syscall"
	"os/exec"
	"os/signal"

	sd "github.com/nathanaelle/sdialog"
)


func valid_file(file string) error {
	st,err	:= os.Stat(file)
	if err != nil {
		return	err
	}

	mode	:= st.Mode()
	if !mode.IsRegular() || (mode.Perm()&0111) == 0 {
		return	fmt.Errorf("%s : no exec permission", file)
	}
	return	nil
}



func exec_cmd(cmd *exec.Cmd, end,reload <-chan struct{}, oups chan<- struct{}, wg *sync.WaitGroup) {
	defer	cmd.Stdout.(io.WriteCloser).Close()
	defer	cmd.Stderr.(io.WriteCloser).Close()

	wg.Add(1)
	defer wg.Done()

	if err := cmd.Start(); err != nil {
		sd.SD_ERR.Error(err)
		oups<-struct{}{}
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
	case	<-reload:
		stop_cmd(cmd)
		sd.Notify(sd.Reloading())

	case	<-end:
		exiting = true
		stop_cmd(cmd)
		sd.Notify(sd.Stopping())

	case	<-exited:
		if !exiting {
			if !cmd.ProcessState.Exited() {
				sd.SD_ALERT.Log("WTF ! exit signal from an non terminated process")
			}
			sd.Notify(sd.Stopping())
			oups<-struct{}{}
		}
	}
}



func	wait_kill(cmd *exec.Cmd, sig syscall.Signal) <-chan struct{} {
	c	:= make(chan struct{})
	go func(){
		cmd.Process.Signal(sig)
		cmd.Process.Wait()
		close(c)
	}()
	return c
}


func	try_kill(cmd *exec.Cmd, sig syscall.Signal) bool {
	sd.Logf(sd.SD_DEBUG, "%s : send %s", cmd.Path, sig.String() )

	select {
	case	<-wait_kill(cmd, sig):
	case	<-time.After(500*time.Millisecond):
	}

	return cmd.ProcessState.Exited()
}



func	stop_cmd(cmd *exec.Cmd) {
	if(try_kill(cmd, syscall.SIGTERM)){
		return
	}

	if(try_kill(cmd, syscall.SIGINT)){
		return
	}

	if(try_kill(cmd, syscall.SIGKILL)){
		return
	}

	try_kill(cmd, syscall.SIGABRT)
}


func	SignalCatcher(wg *sync.WaitGroup) (<-chan struct{},<-chan struct{})  {
	end	:= make(chan struct{})
	reload	:= make(chan struct{})

	go func() {
		wg.Add(1)
		defer wg.Done()

		signalChannel	:= make(chan os.Signal)

		signal.Notify(signalChannel, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		defer close(signalChannel)
		defer close(reload)
		defer close(end)

		for sig := range signalChannel {
			switch	sig {
			case	syscall.SIGHUP:
				reload <- struct{}{}
			case	syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM:
				return
			}
		}
	}()

	return end,reload
}
