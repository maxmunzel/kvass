# kvass: a personal key-value store

![kvass_small](https://user-images.githubusercontent.com/5411096/179968508-5fe1e390-3136-46a6-bb1e-8d329ad231c3.jpeg)


```bash
# simple usage
$ kvass set hello world
$ kvass get hello
world

# enumerate keys
$ kvass ls
hello

# store arbitrary files
$ kvass set logo < kvass.jpg
$ kvass get logo > kvass.jpg


# Its trivial to set up and operate kvass across multiple devices
$ ssh you@yourserver.com kvass config show

Encryption Key:  	5abf59f5f1a2f3c998a4f592ce081a23e14a68fd8a792259c6ec0fc1e8fb1246  # <- copy this for the next step
ProcessID:       	752176921
Remote:          	(None)

$ kvass config key 5abf59f5f1a2f3c998a4f592ce081a23e14a68fd8a792259c6ec0fc1e8fb1246 # set the same key for all your devices
$ kvass config remote yourserver.com:8000 # tell kvass where to find the server instance

# run "kvass serve" on your server using systemd, screen or the init system of your choice. (runit, anyone?)

# every set will now be broadcasted to the server
$ kvass set "hello from the other side" hello
$ ssh you@yourserver kvass get "hello from the other side"
hello

# and every get will check the server for updates
$ ssh you@yourserver kvass set hello 👋
$ kvass get hello
👋

# Good to know: All communication between the client and server is authenticated and encrypted using AES-256 GCM.

# remember the file we stored earlier? Let's get a shareable url for it!
$ kvass url logo
http://demo.maxnagy.com:8000/get?q=OQMwTQmFCz6xiWxFxt4Mkw

# you can also print the corresponding qr code directly to your terminal
kvass qr logo
```
![Screen Shot 2022-07-20 at 13 23 17](https://user-images.githubusercontent.com/5411096/179970204-f1034add-ce07-4f40-b279-0ac25969c069.png)

```
# run kvass without arguments to get a nice cheat sheet of supported commands
$ kvass
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
    kvass url      show shareable url of an entry
    kvass qr       print shareable qr code of entry to console
    kvass config   set config parameters
    kvass serve    start in server mode
```

# Installation

```bash
go install github.com/maxmunzel/kvass@latest
```

# How Syncing works

TL;DR There is a central server running `kvass serve` with clients
connected to it. Key-value pairs overwrite each other based on wall
clock time 99.999% of the time and using Lamport clocks in the
remaining .001% of the time. You can mostly forget about this, as long
as your clocks are mostly in sync.

Let's dive into the details!

Each time we `set` or `rm` a key, kvass creates a new `KvEntry` struct and
merges it onto its local state. The local state is a set of `KvEntry`s that
represent a key-value mapping.
Technically, `kvass rm key` is `kvass set key ""`, so they work the same way.

`KvEntry` is defined as follows:
```go
type KvEntry struct {
	Key      string
	Value    []byte // empty slice means key deleted
	UrlToken string // random token used for url

	// The following fields are used for state merging
	TimestampUnixMicro int64
	ProcessID          uint32 // randomly chosen for each node
	Counter            uint64 // lamport clock
}
```

## Lamport Clocks

Lamport Clocks are a common and easy way to order events in a distributed system.
The [Wikipedia](https://en.wikipedia.org/wiki/Lamport_timestamp) summarizes them nicely:
>   The algorithm follows some simple rules:
>   1. A process increments its counter before each local event (e.g., message sending event);
>   2. When a process sends a message, it includes its counter value with the message after executing step 1;
>   3. On receiving a message, the counter of the recipient is updated, if necessary, to the greater of its current counter and the timestamp in the received message. The counter is then incremented by 1 before the message is considered received.

In kvass, sending a message means `set`ting a key locally and
receiving a message means merging a `KvEntry` into the local state.

Lamport clocks have a nice property: If an event `a` happens causally
after another event `b`, then it follows, that `a.count` > `b.count`:


```
node1     set foo=bar   ->   set foo=baz    \  (send KvEntry{Key="foo", Counter=2, Value="baz"} to node2)
count=0   count = 1          count = 2       \
                                              \
node2                                           ->   rm foo
count=0                                              count = 3
```

node2 updated its counter upon receiving node1's update, so the counter values nicely reflect the fact,
that node2 knew about node1's updates to foo before deleting it. This isn't always helpful though:

```
node1     set foo=bar     \  (send KvEntry{Key="foo", Counter=1, Value="bar"} to node2)
count=0   count = 1        \
                            \
node2     set foo=baz         >  (What is foo supposed to be know?)
count=0   count = 1
```

The canonical answer in this case is to resolve conflicts based on node number (`ProcessID`).
This introduces a global, consistent and total order of events. However it may not always reflect
the order in which a user actually performs the events
```
node1     set foo=bar   ->   set foo=baz                      \  (send KvEntry{Key="foo", Counter=2, Value="baz"} to node2)
count=0   count = 1          count = 2                         \
                                                                \
node2                                         rm foo             ->   (foo is not set to "baz" again)
count=0                                       count = 1
```

In kvass, we therefore use wall-clock time to resolve conflicts: The most recent action of a user is probably the
one he intents to persist, independent of the node he triggered it on. Still, we use Lamport timestamps to handle
identical timestamps and to keep track of which `KvEntry`s we need to exchange between nodes:


```
node1     set foo=bar   ->   set foo=baz                      \  (send KvEntry{Key="foo", Counter=2, Value="baz"} to node2)
count=0   count = 1          count = 2                         \
                                                                \
node2                                         rm foo             ->   (foo stays removed, its
count=0                                       count = 1               count increased to 3)
```

The merging of states is actually trivial now:
```go
func (s *SqlitePersistance) UpdateOn(entry KvEntry) error {
	
    oldEntry := getCurrentEntryFromDB(entry.Key)

    // update the remote counter
    s.State.RemoteCounter = mathutil.MaxUint64(s.State.RemoteCounter, entry.Counter)

    // select LUB of old and new entry
    entry = entry.Max(oldEntry) // returns the entry with the greater (time, counter, - pid) tuple

    // update local counter
    newCounter := mathutil.MaxUint64(entry.Counter, s.State.Counter) + 1
    s.State.Counter = newCounter

    // set new entries counter
    entry.Counter = newCounter

    // write back LUB to db
}
```

`LUB` is CRDT-speak for "least upper bound" ie the smallest entry that is >= the old and new entry.
We convince ourself, that the `KvEntry.Max()` satisfies this LUB property and can therefore derive
that it is also commutative, *idempotent* and associative.




# Shoutouts

[Charm skate](https://github.com/charmbracelet/skate) -- the inspiration for this tool

