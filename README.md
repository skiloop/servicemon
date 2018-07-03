servicemon
========
Service Monitor, a tool to guard service, automatic check availability of a service and restart or switch to alternative

### usage
```shell
Usage: servicemon [--verbose] [--foreground] [--restart] [--restart-delay RESTART-DELAY] [--output OUTPUT] [--env ENV] [--checker CHECKER] [--interval INTERVAL] [--limit LIMIT] [--delay DELAY] [--result RESULT] [--secondary-cmd SECONDARY-CMD] [--secondary-options SECONDARY-OPTIONS] PRIMARYCMD [OPTIONS [OPTIONS ...]]

Positional arguments:
  PRIMARYCMD             primary command
  OPTIONS                primary command options

Options:
  --verbose, -v          verbose
  --foreground, -f       work in foreground
  --restart, -r          restart after instance exit
  --restart-delay RESTART-DELAY, -d RESTART-DELAY
                         restart delay, example: 1s
  --output OUTPUT, -o OUTPUT
                         output file
  --env ENV, -e ENV      additional env for service, multiple values, 
  --checker CHECKER, -c CHECKER
                         script to check if process is healthy, if not healthy then program will stop
  --interval INTERVAL, -i INTERVAL
                         checker interval, example: 1s
  --limit LIMIT, -l LIMIT
                         set open files limit
  --delay DELAY, -D DELAY
                         checker delay after service start, example: 1s
  --result RESULT, -R RESULT
                         healthy checker result
  --secondary-cmd SECONDARY-CMD, -s SECONDARY-CMD
                         secondary command, secondary will start when primary service is not healthy 
  --secondary-options SECONDARY-OPTIONS, -O SECONDARY-OPTIONS
                         secondary options, if secondary command is not set then this is the secondary options for primary command
  --help, -h             display this help and exit
  --version              display version and exit
```

### environments

program env variable can be set with -e options or via servicemon env variable

examples:

```shell
servicemon -e abc=XXXX program
```

or 

```shell
abc=XXX servicemon program
```