# factorialsucks

ALL CREDITS FOR THE ORIGINAL SCRIPT GO TO [alejoar](https://github.com/alejoar/factorialsucks)

## Updates on this FORK

- The app now allows setting **environment variables** for **credentials** (*FACTORIAL_EMAIL*, *FACTORIAL_PASS*)
- Now is possible to define **multiple clock-in and clock-out** (--ci xxx --ci xxx ...)
- It uses a **different time** for **Fridays** (with a really weird and dirty harcoded workaround)

## Requirements for building

:warning: You need to have **go environment** installed to be able to run and build the app.

### On Ubuntu

```bash
sudo apt install golang-go
```

### On MacOs

...come on. Just switch to a real OS.

## Build & install

```bash
go build -o build/factorialsucks factorialsucks.go utils.go
```

Then move the binary to a more convenient path:

```bash
sudo mv build/factorialsucks /usr/local/bin
```

## Setup credentials

As an upgrade to the original code you can now use **environment variables**!

This allows to achieve **full automation** :-)

```bash
export FACTORIAL_EMAIL='xxxx@example.com'
export FACTORIAL_PASSWORD='xxxx'
```

If you don't provide the environment variables the script will still ask you to input the values manually just like before.

## Show help

```
❯ factorialsucks -h

NAME:
   factorialsucks - FactorialHR auto clock in for the whole month from the command line

USAGE:
   factorialsucks [options]

GLOBAL OPTIONS:
   --email value, -e value        you factorial email address
   --year YYYY, -y YYYY           clock-in year YYYY (default: current year)
   --month MM, -m MM              clock-in month MM (default: current month)
   --clock-in HH:MM, --ci HH:MM   clock-in time HH:MM. You can define it multiple times.
   --clock-out HH:MM, --co HH:MM  clock-in time HH:MM. You can define it multiple times.
   --today, -t                    clock in for today only (default: false)
   --until-today, --ut            clock in only until today (default: false)
   --dry-run, --dr                do a dry run without actually clocking in (default: false)
   --reset-month, --rm            delete all shifts for the given month (default: false)
   --help, -h                     show help (default: false)
```

## How to run

The following example will clock everyday from 09:00 to 14:00 **except fridays** that will clock 09:00 to 15:00. **This is hardcoded in the code at the moment.**. You still have to supply the clock-in and clock-out hours for the rest of the days.

```bash
factorialsucks --ci 09:00 --co 14:00
```

As an update to the original script, you can now set **multiple clock-in and clock-out**!

```bash
factorialsucks --ci 09:00 --co 14:00 --ci 15:00 --co 18:30
```
> Note that providing a clock-in later than 15:00 in the command line, will make **friday** to clock a **second row**. This should be avoided, I'll might fix it in the future.

To clock only until today, just use the --ut argument:

```bash
factorialsucks --ci 09:00 --co 14:00 --ci 15:00 --co 18:30 --ut
```

If you wish to **remove all entries** for the current month, run:

```bash
factorialsucks --reset-month
```

## Running in a cron

Edit your local crontab:

```bash
crontab -e
```

Add at the end:

```bash
00 19   * * *   root    echo "Starting..."; FACTORIAL_EMAIL='xxxx@example.com' FACTORIAL_PASSWORD='xxxx' factorialsucks --ci 09:00 --co 14:00 --ci 15:00 --co 18:30 --ut >> $HOME/factorial.txt; echo -e "\nFinished at: $(date)\n\n---" >> $HOME/factorial.txt;
```

> Note that you should be able to load the environment variables from a file, for example *bash_profile* or *bashrc*
