package	main

import	(
	"os"
	"fmt"
	"flag"
	"path"
	"sync"
	"time"
	"strings"
	"syscall"
	"os/exec"
	"os/signal"
	"crypto/sha512"
)


func main() {
	logdir	:= flag.String("logdir", "/tmp", "path where the log will be stored")

	flag.Parse()

	wg	:= new(sync.WaitGroup)
	args	:= flag.Args()
	filexec	:= path.Clean(args[0])
	basexec	:= path.Base(filexec)

	stat,err:= os.Stat(filexec)
	if err != nil {
		panic(err)
	}

	mode := stat.Mode()

	if !mode.IsRegular() || (mode.Perm()&0111) == 0 {
		panic("no access to : "+ filexec)
	}

	count	:= 0
	oups	:= make(chan bool)

	end,reload	:= SignalCatcher(wg)
	sd_watchdog(end,wg)

	args = append([]string{ filexec }, args[1:]... )
	last	:= time.Now()

	for {
		go exec_cmd(last, count, path.Join(*logdir,basexec), filexec, args, end,reload,oups,wg)

		select {
		case <-end:
			return

		case <-oups:
			count+=1
			if time.Now().Sub(last) < time.Minute {
				time.Sleep(time.Minute)
			}
			last = time.Now()
		}
	}

	wg.Wait()
}


func exec_cmd(now time.Time, count int, baselog, filexec string, args []string, end,reload <-chan bool, oups chan<- bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	ts	:= fmt.Sprintf("%05d-%4d%02d%02dT%02d%02dZ", count, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute() )

	stdout	:= NewHashedWriteCloser(strings.Join([]string{ baselog, ts, "out", "log"}, "."), sha512.New() )
	stderr	:= NewHashedWriteCloser(strings.Join([]string{ baselog, ts, "err", "log"}, "."), sha512.New() )
	defer	stdout.Close()
	defer	stderr.Close()

	cmd	:= &exec.Cmd {
		Path:	filexec,
		Args:	args,
		Stdin:	nil,
		Stdout:	stdout,
		Stderr: stderr,
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	sd_notify("READY","1")

	exited := make(chan bool,1)
	go func() {
		cmd.Wait()
		close(exited)
	}()

	exiting := false
	select {
	case	<-reload:
		stop_cmd(filexec, cmd )
		sd_notify("RELOADING","1")

	case	<-end:
		exiting = true
		stop_cmd(filexec, cmd )
		sd_notify("STOPPING","1")

	case	<-exited:
		if !exiting {
			if !cmd.ProcessState.Exited() {
				panic("WTF ! exit signal from an non terminated process")
			}
			sd_notify("RELOADING","1")
			sd_logf("%s : %s", filexec, "terminated - reload")
			oups <- true
		}
	}
}


func	stop_cmd(filexec string, cmd *exec.Cmd) {
	sd_logf("%s : %s", filexec, "send SIGTERM")
	cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(500*time.Millisecond)

	if !cmd.ProcessState.Exited() {
		sd_logf("%s : %s", filexec, "send SIGKILL")
		cmd.Process.Signal(syscall.SIGKILL)
	}

}


func	SignalCatcher(wg *sync.WaitGroup) (<-chan bool,<-chan bool)  {
	end	:= make(chan bool,1)
	reload	:= make(chan bool,1)

	go func() {
		wg.Add(1)
		defer wg.Done()

		signalChannel	:= make(chan os.Signal,1)

		signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		defer close(signalChannel)
		defer close(reload)
		defer close(end)

		for sig := range signalChannel {
			switch	sig {
			case	syscall.SIGHUP:
				reload <- true
			case	syscall.SIGINT, syscall.SIGTERM:
				return
			}
		}
	}()

	return end,reload
}
