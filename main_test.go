package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
)

func TestMain(m *testing.M) {
	logging.SetLogLevel("*", "error")
	// logging.SetLogLevel("store", "debug")
	// logging.SetLogLevel("threads", "debug")
	// logging.SetLogLevel("threadstore", "debug")
	os.Exit(m.Run())
}

func TestSingleUser(t *testing.T) {
	t.Parallel()
	c1, clean1 := createClient(t, "user1")
	defer clean1()
	defer c1.Close()
	err := c1.Start()
	checkErr(t, err)
	invlink, err := c1.InviteLink()
	checkErr(t, err)

	if invlink == "" {
		t.Fatalf("invite link can't be empty")
	}

	trees, err := c1.GetDirectoryTree()
	checkErr(t, err)
	if len(trees) != 1 {
		t.Fatalf("there should be one user folder")
	}
	tree := trees[0]
	if tree.Owner != "user1" || tree.ID == "" || len(tree.Files) != 0 {
		t.Fatalf("invalid initial tree")
	}

	tmpFilePath := path.Join(c1.sharedFolderPath, "user1", "test.txt")
	f, err := os.OpenFile(tmpFilePath, os.O_RDWR|os.O_CREATE, 0660)
	checkErr(t, err)
	defer os.Remove(tmpFilePath)
	_, err = f.Write([]byte("This is some content for the file"))
	checkErr(t, err)
	checkErr(t, f.Close())

	time.Sleep(time.Second)
	trees, err = c1.GetDirectoryTree()
	checkErr(t, err)
	if len(trees) != 1 {
		t.Fatalf("there should be one user folder")
	}
	tree = trees[0]
	if len(tree.Files) != 1 || tree.Files[0].Name != tmpFilePath {
		t.Fatalf("invalid tree state")
	}
}

func TestNUsersBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		totalClients   int
		totalCorePeers int
		syncTimeout    time.Duration
	}{
		{totalClients: 2, totalCorePeers: 1, syncTimeout: time.Second * 3},
		{totalClients: 5, totalCorePeers: 1, syncTimeout: time.Second * 5},
		{totalClients: 10, totalCorePeers: 1, syncTimeout: time.Second * 10},
		{totalClients: 10, totalCorePeers: 3, syncTimeout: time.Second * 10},
		{totalClients: 25, totalCorePeers: 5, syncTimeout: time.Second * 20},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Total%dCore%d", tt.totalClients, tt.totalCorePeers), func(t *testing.T) {
			t.Parallel()
			var clients []*Client

			for i := 0; i < tt.totalClients; i++ {
				c, clean := createClient(t, fmt.Sprintf("user%d", i))
				defer clean()
				defer c.Close()
				clients = append(clients, c)
			}
			err := clients[0].Start()
			checkErr(t, err)
			invlink0, err := clients[0].InviteLink()
			checkErr(t, err)
			for i := 1; i < tt.totalCorePeers; i++ {
				checkErr(t, clients[i].StartFromInvitation(invlink0))
			}

			for i := tt.totalCorePeers; i < tt.totalClients; i++ {
				rotatedInvLink, err := clients[i%tt.totalCorePeers].InviteLink()
				checkErr(t, err)
				checkErr(t, clients[i].StartFromInvitation(rotatedInvLink))

			}
			time.Sleep(tt.syncTimeout)

			assertClientsEqualTrees(t, clients)
		})
	}

}

func assertClientsEqualTrees(t *testing.T, clients []*Client) {
	totalClients := len(clients)
	dtrees := make([][]*sharedFolder, totalClients)
	for i := range clients {
		tree, err := clients[i].GetDirectoryTree()
		checkErr(t, err)
		dtrees[i] = tree
	}
	if !EqualTrees(totalClients, dtrees...) {
		for i := range dtrees {
			printTree(i, dtrees[i])
		}
		t.Fatalf("trees from users aren't equal")
	}
}

func printTree(i int, folders []*sharedFolder) {
	sort.Slice(folders, func(i, j int) bool {
		return strings.Compare(folders[i].Owner, folders[j].Owner) < 0
	})

	fmt.Printf("Tree of user %d\n", i)
	for _, sf := range folders {
		fmt.Printf("\t%s %s\n", sf.ID, sf.Owner)
		for _, f := range sf.Files {
			fmt.Printf("\t\t %s %s\n", f.Name, f.CID)
		}
	}
	fmt.Println()
}

func EqualTrees(numUsers int, trees ...[]*sharedFolder) bool {
	base := trees[0]
	if len(base) != numUsers {
		return false
	}
	for i := 1; i < len(trees); i++ {
		if len(base) != len(trees[i]) {
			return false
		}
		for _, folder := range base {
			for _, folder2 := range trees[i] {
				if folder2.ID == folder.ID && folder2.Owner == folder.Owner {
					if !EqualFileList(folder.Files, folder2.Files) {
						return false
					}
				}
			}
		}
	}
	return true
}

func EqualFileList(f1s, f2s []file) bool {
	if len(f1s) != len(f2s) {
		return false
	}
	for _, f := range f1s {
		exist := false
		for _, f2 := range f2s {
			if f.ID == f2.ID {
				if !EqualFiles(f, f2) {
					return false
				}
				exist = true
				break
			}
		}
		if !exist {
			return false
		}
	}
	return true
}

func EqualFiles(f, f2 file) bool {
	if f.Name != f2.Name || f.IsDirectory != f2.IsDirectory ||
		f.CID != f2.CID || len(f.Files) != len(f2.Files) {
		return false
	}
	for _, ff := range f.Files {
		exist := false
		for _, ff2 := range f2.Files {
			if ff.ID == ff2.ID {
				if !EqualFiles(ff, ff2) {
					return false
				}
				exist = true
				break
			}
		}
		if !exist {
			return false
		}
	}
	return true
}

func createClient(t *testing.T, name string) (*Client, func()) {
	shrFolder, err := ioutil.TempDir("", "")
	checkErr(t, err)
	repoPath, err := ioutil.TempDir("", "")
	checkErr(t, err)
	client, err := NewClient(name, shrFolder, repoPath)
	checkErr(t, err)
	return client, func() {
		os.RemoveAll(shrFolder)
		os.RemoveAll(repoPath)
	}
}

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
