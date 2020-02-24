package main

import (
	"crypto/sha512"
	"flag"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	sd "github.com/nathanaelle/sdialog/v2"
)

func main() {
	logdir := flag.String("logdir", "/tmp", "path where the log will be stored")

	flag.Parse()

	oups := make(chan struct{})

	count := 0
	wg := new(sync.WaitGroup)
	ctx, reload := signalCatcher(wg)
	sd.Watchdog(ctx, wg)

	args := flag.Args()
	filexec := path.Clean(args[0])
	basexec := path.Base(filexec)
	baselog := path.Join(*logdir, basexec)

	err := validFile(filexec)
	for err != nil {
		sd.LogALERT.LogError(err)
		time.Sleep(time.Minute)
		err = validFile(filexec)
	}

	args = append([]string{filexec}, args[1:]...)

	for {
		start := time.Now()
		ts := fmt.Sprintf("%05d-%4d%02d%02dT%02d%02dZ", count, start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute())

		go execCmd(ctx, &exec.Cmd{
			Path:   filexec,
			Args:   args,
			Stdin:  nil,
			Stdout: newHashedWriteCloser(strings.Join([]string{baselog, ts, "out", "log"}, "."), sha512.New()),
			Stderr: newHashedWriteCloser(strings.Join([]string{baselog, ts, "err", "log"}, "."), sha512.New()),
		}, reload, oups, wg)

		select {
		case <-ctx.Done():
			break

		case <-oups:
			count++
			if time.Now().Sub(start) < time.Minute {
				time.Sleep(time.Minute)
			}
		}
	}

	wg.Wait()
}
