package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/google/uuid"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-cid"
	"github.com/jsign/threads-fw/watcher"
	core "github.com/textileio/go-textile-core/store"
	es "github.com/textileio/go-textile-threads/eventstore"
)

type Client struct {
	cancel           context.CancelFunc
	store            *es.Store
	model            *es.Model
	watcher          *watcher.FolderWatcher
	peer             *ipfslite.Peer
	fetchedCID       map[cid.Cid][]byte
	sharedFolderPath string
	name             string
	myFolder         *sharedFolder
}

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

func (c *Client) Close() error {
	c.store.Close()
	c.watcher.Close()
	c.cancel()

	return nil
}

func NewClient(name, sharedFolderPath, repoPath string) (*Client, error) {
	ts, err := es.DefaultThreadservice(repoPath, es.ListenPort(0), es.ProxyPort(0))
	if err != nil {
		return nil, err
	}
	//ts.Bootstrap(util.DefaultBoostrapPeers())
	s, err := es.NewStore(ts, es.WithRepoPath(repoPath))
	if err != nil {
		return nil, fmt.Errorf("error when creating store: %v", err)
	}

	go func() {
		l := s.StateChangeListen()
		for range l.Channel() {

		}
	}()
	m, err := s.Register("shardFolder", &sharedFolder{})
	if err != nil {
		return nil, fmt.Errorf("error when registering model: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ipfspeer, err := createIPFSLite(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error when creating ipfs lite peer: %v", err)
	}

	return &Client{
		cancel:           cancel,
		store:            s,
		model:            m,
		peer:             ipfspeer,
		fetchedCID:       make(map[cid.Cid][]byte),
		sharedFolderPath: sharedFolderPath,
		name:             name,
	}, nil
}

func (c *Client) Start() (string, error) {
	log.Info("Starting a new thread for shared folders")
	if err := c.store.Start(); err != nil {
		return "", err
	}
	inviteLink, err := generateInviteLink(c.store)
	if err != nil {
		return "", err
	}

	if err = c.bootstrap(); err != nil {
		return "", err
	}

	return inviteLink, nil
}

func (c *Client) StartFromInvitation(link string) error {
	addr, fk, rk := parseInviteLink(link)
	log.Infof("Starting from addr: %s", addr)
	if err := c.store.StartFromAddr(addr, fk, rk); err != nil {
		return err
	}

	return c.bootstrap()
}

func (c *Client) bootstrap() error {
	myFolderPath := path.Join(c.sharedFolderPath, c.name)
	myFolder, err := getOrCreateMyFolderInstance(c.model, myFolderPath, c.name)
	if err != nil {
		return fmt.Errorf("error when getting client folder instance: %v", err)
	}

	w, err := watcher.New(myFolderPath, c.onCreate)
	if err != nil {
		return fmt.Errorf("error when creating folder watcher: %v", err)
	}
	w.Watch()

	c.watcher = w
	c.myFolder = myFolder
	return nil
}

func (c *Client) Listen() *es.StateChangeListener {
	return c.store.StateChangeListen()
}

func (c *Client) GetDirectoryTree() ([]*sharedFolder, error) {
	var res []*sharedFolder
	if err := c.model.Find(&res, nil); err != nil {
		return nil, err

	}
	return res, nil
}

func (c *Client) ensureFiles() error {
	res, err := c.GetDirectoryTree()
	if err != nil {
		return err
	}
	for _, owner := range res {
		for _, f := range owner.Files {
			c.ensureCID(owner.Owner, f.Name, f.CID)
		}
	}
	return nil
}

func (c *Client) ensureCID(owner, name, cidStr string) error {
	cid, err := cid.Decode(cidStr)
	if err != nil {
		return err
	}
	if _, ok := c.fetchedCID[cid]; ok {
		return nil
	}
	str, err := c.peer.GetFile(context.Background(), cid)
	if err != nil {
		return err
	}
	savePath := path.Join(c.sharedFolderPath, owner, name)
	f, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, str); err != nil {
		return err
	}
	return nil
}

func getOrCreateMyFolderInstance(m *es.Model, myFolderPath, name string) (*sharedFolder, error) {
	if _, err := os.Stat(myFolderPath); os.IsNotExist(err) {
		if err = os.MkdirAll(myFolderPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	var res []*sharedFolder
	if err := m.Find(&res, es.Where("Owner").Eq(name)); err != nil {
		return nil, err
	}

	var myFolder *sharedFolder
	if len(res) == 0 {
		ownFolder := &sharedFolder{Owner: name, Files: []file{}}
		if err := m.Create(ownFolder); err != nil {
			return nil, err
		}
		myFolder = ownFolder
		// fmt.Printf("####### I %s have entityid %s\n", name, ownFolder.ID.String())
	} else {
		myFolder = res[0]
	}

	return myFolder, nil
}

func (c *Client) onCreate(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	n, err := c.peer.AddFile(context.Background(), f, nil)
	if err != nil {
		return err
	}
	newFile := file{ID: uuid.New().String(), Name: fileName, CID: n.Cid().String(), Files: []file{}}
	c.myFolder.Files = append(c.myFolder.Files, newFile)
	return c.model.Save(c.myFolder)

}
