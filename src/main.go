package	main

import	(
	"fmt"
	"flag"
	"path"
	"sync"
	"time"
	"strings"
	"os/exec"
	"crypto/sha512"

	sd "github.com/nathanaelle/sdialog"
)


func main() {
	logdir	:= flag.String("logdir", "/tmp", "path where the log will be stored")

	flag.Parse()

	count		:= 0
	oups		:= make(chan struct{})
	wg		:= new(sync.WaitGroup)
	end,reload	:= SignalCatcher(wg)
	sd.Watchdog(end, wg)


	args	:= flag.Args()
	filexec	:= path.Clean(args[0])
	basexec	:= path.Base(filexec)
	baselog	:= path.Join(*logdir,basexec)

	err	:= valid_file(filexec)
	for	err != nil {
		sd.SD_ALERT.Error(err)
		time.Sleep(time.Minute)
		err = valid_file(filexec)
	}

	args	= append([]string{ filexec }, args[1:]... )

	for {
		start	:= time.Now()
		ts	:= fmt.Sprintf("%05d-%4d%02d%02dT%02d%02dZ", count, start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute() )

		go	exec_cmd(&exec.Cmd {
			Path:	filexec,
			Args:	args,
			Stdin:	nil,
			Stdout:	NewHashedWriteCloser(strings.Join([]string{ baselog, ts, "out", "log"}, "."), sha512.New() ),
			Stderr: NewHashedWriteCloser(strings.Join([]string{ baselog, ts, "err", "log"}, "."), sha512.New() ),
		}, end, reload, oups, wg)

		select	{
		case	<-end:
			break

		case	<-oups:
			count+=1
			if time.Now().Sub(start) < time.Minute {
				time.Sleep(time.Minute)
			}
		}
	}

	wg.Wait()
}
