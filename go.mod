module github.com/jsign/threads-fw

go 1.13

require (
	cloud.google.com/go v0.38.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/google/uuid v1.1.1
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/hsanjuan/ipfs-lite v0.1.7
	github.com/ipfs/go-bitswap v0.1.8
	github.com/ipfs/go-cid v0.0.3
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-graphsync v0.0.3
	github.com/ipfs/go-ipfs-blockstore v0.1.0
	github.com/ipfs/go-log v0.0.1
	github.com/libp2p/go-libp2p v0.4.0
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/libp2p/go-libp2p-routing v0.1.0
	github.com/mr-tron/base58 v1.1.2
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/textileio/go-eventstore v0.0.0-20191106220529-2723bb6c7c79
	github.com/textileio/go-textile-core v0.0.0-20191119181245-af71494bbb10
	github.com/textileio/go-textile-threads v0.0.0-20191120233028-2d227e65ef91
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc
	golang.org/x/sys v0.0.0-20191113165036-4c7a9d0fe056 // indirect
	google.golang.org/api v0.14.0 // indirect
	gopkg.in/src-d/go-log.v1 v1.0.1
)

replace github.com/textileio/go-textile-threads v0.0.0-20191120233028-2d227e65ef91 => ../go-textile-threads
