# Folder-sync app
This is stil a WIP, but here's a general idea of the intention.

It allows any number of peers to join to maintain a _shared folder_ where everyone 
of them has their own folder.

This is modeled by a unique `Model` with the following struct:
```
type sharedFolder struct {
	ID    core.EntityID
	Owner string
	Files []file
}

type file struct {
	ID   string
	Name string
	CID  string

	IsDirectory bool
	Files       []file
}
```

Where `sharedFolder` is one instance per peer that joined the `Thread`. There we can 
see an `ID` which is the identity, `Owner` which is the name of the peer, and a slice 
of `file`, which is a recursive data-structure to model a directory structure.

A `file` can be a raw single file or a directory. If is a raw file, it has a `CID` which 
will be fetched offband by some strategy (options: p2p between thread peers, ipfs, filecoin).

Currently, when a peer joins the app, it will create his own shared folder instance (since 
every peer owns one). Others will discoverd him via `Thread` syncing.

There's also an on-progress filewatcher, which will automatically discover dropped files in 
the folder and add them automatically to the model to keep things _Dropbox_ style.

As discussed internally, we can abstract the CID fetching strategy to be modular so we can play 
with the different options mentioned.

Eventually, will be nice to enable a WS endpoint to add a web UI interface to let interact with 
the app too.

Peers join each other via an auto-generated link invite with format: `<thread-addr>?<follow-key>&<read-key>`

## Tests
This app *now* has a pretty heavy test setup, where tunning a parameter simulates N peers joining 
the thread and syncing. After some specified time, stores from all peers are compared to assert 
that all converged to the same state (all peers know about all other peers, with exactly all their 
files metadata).

### Run all tests
`go test ./...`

### E2E heavy test
To run this tests in particular: `go test ./... -run TestNUserBootstrap`.
The default setup are 5 peers joining the peer that created the thread (peer0). So this peer is 
heavily bombarded with joining peers. Eventually they should start to know about others and continue 
syncing.

If you want to play with heavy loads, you can edit the `totalClients := 5` with whatever number you 
prefer, and enjoy getting a feeling how things behave. Peers communicate only via network interface, 
no shared stated by files or anything similar. Also, you can play with the `time.Sleep()` time of 
syncing before asserting Store convergence between peers.

Both params can be played with to get a feeling how things work. Currently, these tests forced some 
hard concurrency cuts in `go-textile-threads` since some state-fatal bugs existed. So we can assume 
current performance is the worst. It would be very important to always test `go-textile-threads` changes 
under this heavy test since is the currently most complete and heavy one.

Further tests will make these multiple peers adding files and having long running syncs to assert 
convergence in store state. All this is to ensure proper functioning of `Store` and `ThreadsV2`. 
Most of the fetching CID work is completely independent since this is offloaded from Threads; these 
tests will exist when working in that line soon.
