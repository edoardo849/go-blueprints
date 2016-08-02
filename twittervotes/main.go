package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"
	"github.com/joeshaw/envdecode"
	"gopkg.in/mgo.v2"
)

func main() {
	var stoplock sync.Mutex
	stop := false

	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		// wait-block until a signal is sent through
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("Stopping...")
		stopChan <- struct{}{}
		closeConn()
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	if err := dialdb(); err != nil {
		log.Fatalln("failed to dial MongoDB: ", err)
	}
	defer closedb()

	votes := make(chan string)
	publisherStoppedChan := publishVotes(votes)

	twitterStoppedChan := startTwitterStream(stopChan, votes)
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				break
			}
			stoplock.Unlock()
		}
	}()
	<-twitterStoppedChan
	close(votes)
	<-publisherStoppedChan

	fmt.Println("hello")
}

func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)
	pub, _ := nsq.NewProducer("192.168.99.100:4150", nsq.NewConfig())

	go func() {

		// continue to pull values from the channel. When the channel has no
		// values the execution will be blocked until a values comes down again.
		// If the votes channel is closed, the loop will exit.
		// https://www.youtube.com/watch?v=SmoM1InWXr0
		for vote := range votes {
			pub.Publish("votes", []byte(vote)) // publish the vote to the queue
		}
		log.Println("Publisher: Stopping")
		pub.Stop()
		log.Println("Publisher: Stopped")
		stopchan <- struct{}{}
	}()
	return stopchan
}

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string

	// Memory-efficient way to of reading the poll data because it only
	// ever uses a single `poll` object. If we were to use the `All` method
	// instead, the amount of memory we'd use would depend on the number of
	// polls we had in our database, which woulf be out of control.
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll

	for iter.Next(&p) {
		// append is a `variadic` function
		options = append(options, p.Options...)
	}

	// cleanup any used memory on the iterator
	iter.Close()

	return options, iter.Err()
}

// ======
// MongoDB
// ======
var (
	dbSetupOnce sync.Once
	db          *mgo.Session
)

func dialdb() error {
	var err error

	mongoAddr := fmt.Sprintf("%s:%s", dbCreds.Address, dbCreds.Port)
	log.Println("dialing mongodb: ", mongoAddr)

	dbSetupOnce.Do(func() {
		setupdb()
		db, err = mgo.Dial(mongoAddr)
	})

	return err
}

func closedb() {
	db.Close()
	log.Println("closed database connection")
}

var dbCreds struct {
	Port    string `env:"SP_MONGO_PORT,required"`
	Address string `env:"SP_MONGO_ADDRESS,required"`
}

func setupdb() {
	if err := envdecode.Decode(&dbCreds); err != nil {
		log.Fatalln(err)
	}
}
