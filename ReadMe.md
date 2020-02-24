# alignak


## What is alignak ?

alignak is a foreground process watchdog that log stdout and stderr in files, and restart the process if needed.


## Usage

```
alignak -logdir /some/dir -- /usr/local/bin/mattdaemon --opt1 -zkxf foo bar
```

## Features

  * [x] Systemd notify
  * [x] Systemd watchdog
  * [x] Journald alerts
  * [x] SHA2/512 signature logged of the previous logfiles in case of restart
  * [x] custom location of logfiles with -logdir option (default : /tmp )
  * [ ] Restart process in case of binary upgrade with -upgrade
  * [ ] SSKG for message signing

## License

2-Clause BSD

## Todo / wish-list

  * write comments
  * clean some ugly stuff
  * add tests
