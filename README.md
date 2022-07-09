# kvass: A personal key-value store

```bash
# simple usage
$ kvass set hello world
$ kvass get hello
world

# enumerate keys
$ kvass ls
hello

# store arbitrary files
$ kvass set kitty < kitty.jpg
$ kvass get kitty > kitty.jpg

# Its trivial to set up and operate kvass across multiple devices


$ ssh you@yourserver.com kvass config show

Encryption Key:  	5abf59f5f1a2f3c998a4f592ce081a23e14a68fd8a792259c6ec0fc1e8fb1246  # <- copy this for the next step
ProcessID:       	752176921
Remote:          	(None)

$ kvass config key 5abf59f5f1a2f3c998a4f592ce081a23e14a68fd8a792259c6ec0fc1e8fb1246 # set the same key for all your devices
$ kvass config remote yourserver.com:8000

# run "kvass serve" on your server using systemd, screen or the init system of your choice. runit, anyone?

# every set will now be broadcasted to the server

$ kvass set "hello from the other side" hello
$ ssh you@yourserver kvass get "hello from the other side"
hello

# and every get will check the server for updates
$ ssh you@yourserver kvass set hello 👋
$ kvass get hello
👋

# run kvass without arguments to get a nice cheat sheet of supported commands
kvass
kvass [--db=string]

Description:
    kvass - a personal KV store

Options:
        --db       the database file to use (default: ~/.kvassdb.sqlite)

Sub-commands:
    kvass ls       list keys
    kvass get      get a value
    kvass set      set a value
    kvass rm       remove a key
    kvass config   set config parameters
    kvass serve    start in server mode

    kvass serve    start in server mode
```

# Installation

```bash
go install github.com/maxmunzel/kvass@latest
```


# Shoutouts

[Charm skate](https://github.com/charmbracelet/skate) -- the inspiration for this tool  



