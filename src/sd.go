package main


import	(
	"io"
	"os"
	"fmt"
	"net"
	"sync"
	"time"
	"strconv"
)

type	LogLevel	string

const	(
	SD_EMERG	LogLevel = "<0>"	// system is unusable
	SD_ALERT	LogLevel = "<1>"	// action must be taken immediately
	SD_CRIT		LogLevel = "<2>"	// critical conditions
	SD_ERR		LogLevel = "<3>"	// error conditions
	SD_WARNING	LogLevel = "<4>"	// warning conditions
	SD_NOTICE	LogLevel = "<5>"	// normal but significant condition
	SD_INFO		LogLevel = "<6>"	// informational
	SD_DEBUG	LogLevel = "<7>"	// debug-level messages
)

var	(
	systemd_not_running	bool
	systemd_notify_socket	string
	systemd_watchdog_usec	int
)


func	init() {
	var err	error

	systemd_notify_socket = os.Getenv("NOTIFY_SOCKET")

	if systemd_notify_socket == "" {
		systemd_not_running = true
		return
	}

	systemd_watchdog_usec, err = strconv.Atoi(os.Getenv("WATCHDOG_USEC"))
	if err != nil {
		sd_log(SD_ALERT, fmt.Sprintf("WATCHDOG_USEC Error: %s", err.Error()))
	}

}



func	sd_logf(l LogLevel, format string, args ...interface{}) bool {
	if systemd_not_running {
		return false
	}

	return sd_log(l, fmt.Sprintf(format, args...))
}

func	sd_log(l LogLevel, m string) bool {
	if systemd_not_running {
		return false
	}

	io.WriteString(os.Stderr, fmt.Sprintf("%s%s\n",string(l), m))

	return true
}


func sd_watchdog(end <-chan bool, wg *sync.WaitGroup) bool {
	if systemd_not_running {
		return false
	}

	// see http://www.freedesktop.org/software/systemd/man/sd_watchdog_enabled.html
	ticker	:= time.Tick(time.Duration(systemd_watchdog_usec/2) * time.Microsecond)

	go func(){
		wg.Add(1)
		defer wg.Done()

		for {
			select {
			case	<-ticker:
				sd_notify("WATCHDOG", "1")

			case	<-end:
				return
			}
		}
	}()

	return true
}


func sd_notify(state,message string) bool {
	if systemd_not_running {
		return false
	}

	conn, err := net.Dial("unixgram", systemd_notify_socket)
	if err != nil {
		sd_log(SD_ALERT, fmt.Sprintf("NOTIFY_SOCKET Error: %s", err.Error()))
		return false
	}
	defer	conn.Close()

	_, err = conn.Write([]byte(state+"="+message))
	if err != nil {
		return false
	}

	return true
}
