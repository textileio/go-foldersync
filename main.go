package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log"
	logger "github.com/whyrusleeping/go-logging"
)

var (
	log = logging.Logger("main")
)

func main() {
	name := flag.String("name", "guest", "name of the user")
	sharedFolderPath := flag.String("folder", "sharedFolder", "path of the shared folder")
	inviteLink := flag.String("inviteLink", "", "thread addr to join a shared folder")
	debug := flag.Bool("debug", true, "debug mode")
	repoPath := flag.String("repo", "repo", "path of the store repo")
	flag.Parse()

	if *debug {
		logging.SetAllLoggers(logger.ERROR)
		logging.SetLogLevel("main", "debug")
		logging.SetLogLevel("watcher", "debug")
	}

	client, err := NewClient(*name, *sharedFolderPath, *repoPath)
	if err != nil {
		log.Fatalf("error when creating the client: %v", err)
	}

	log.Info("Starting client...")
	if *inviteLink == "" {
		err := client.Start()
		if err != nil {
			log.Fatalf("error when starting peer without invitation: %v", err)
		}
		invLink, err := client.InviteLink()
		if err != nil {
			log.Fatalf("error when generating invitation link: %v", err)
		}
		log.Infof("Invitation link: %s", invLink)
	} else {
		if err := client.StartFromInvitation(*inviteLink); err != nil {
			log.Fatalf("error when starting peer from invitation: %v", err)
		}
	}
	log.Infof("Client started!")

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	// var wg sync.WaitGroup
	// wg.Add(1)
	// l := client.Listen()
	// go func() {
	// 	defer wg.Done()
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case <-l.Channel():
	// 			c.PrintFolderTree()
	// 		}
	// 	}
	// }()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Info("Closing...")
	// cancel()
	// wg.Wait()
	if err = client.Close(); err != nil {
		log.Fatalf("error when closing the client: %v", err)
	}
}
