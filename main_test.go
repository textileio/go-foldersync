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
	invlink, err := c1.Start()
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

func TestTwoUserBootstrap(t *testing.T) {
	t.Parallel()
	c1, clean1 := createClient(t, "user1")
	defer clean1()
	defer c1.Close()
	invlink, err := c1.Start()
	checkErr(t, err)

	time.Sleep(time.Second * 1)

	c2, clean2 := createClient(t, "user2")
	defer clean2()
	defer c2.Close()
	checkErr(t, c2.StartFromInvitation(invlink))

	time.Sleep(time.Second * 2)

	tree1, err := c1.GetDirectoryTree()
	checkErr(t, err)
	tree2, err := c2.GetDirectoryTree()
	checkErr(t, err)

	if !EqualTrees(2, tree1, tree2) {
		t.Fatalf("trees from users aren't equal")
	}
}

func TestNUserBootstrap(t *testing.T) {
	t.Parallel()
	totalClients := 5

	c1, clean1 := createClient(t, "user0")
	defer clean1()
	defer c1.Close()
	invlink, err := c1.Start()
	checkErr(t, err)

	clients := []*Client{c1}
	for i := 1; i <= totalClients-1; i++ {
		c2, clean2 := createClient(t, fmt.Sprintf("user%d", i))
		defer clean2()
		defer c2.Close()
		checkErr(t, c2.StartFromInvitation(invlink))
		clients = append(clients, c2)
	}
	time.Sleep(time.Second * 10)
	dtrees := make([][]*sharedFolder, totalClients)
	for i := range clients {
		dtrees[i], err = clients[i].GetDirectoryTree()
		checkErr(t, err)
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
		defer os.RemoveAll(shrFolder)
		defer os.RemoveAll(repoPath)
	}
}

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
