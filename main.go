package main

import (
	"flag"
	"github.com/jgulick48/lxp-bridge-go/internal/coordinator"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/mitchellh/panicwrap"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var configLocation = flag.String("configFile", "./config.yml", "Location for the configuration file.")

func main() {
	//LogListener()
	startService()
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		// Something went wrong setting up the panic wrapper. Unlikely,
		// but possible.
		panic(err)
	}

	// If exitStatus >= 0, then we're the parent process and the panicwrap
	// re-executed ourselves and completed. Just exit with the proper status.
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}
}

func panicHandler(output string) {
	// output contains the full output (including stack traces) of the
	// panic. Put it in a file or something.
	log.Printf("The child panicked:\n\n%s\n", output)
	os.Exit(1)
}

func startService() {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	logger := logrus.Logger{}
	log.Print(path)
	flag.Parse()
	logrus.Infof("Got config location of %s", *configLocation)
	config := LoadClientConfig(*configLocation)
	done := make(chan bool)
	OnTermination(func() {
		done <- true
	})
	server := coordinator.NewClient(config, &logger)
	for <-done {
		server.Done()
		return
	}
}

// TermFunc defines the function which is executed on termination.
type TermFunc func()

func OnTermination(fn TermFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		select {
		case <-c:
			if fn != nil {
				fn()
			}
		}
	}()
}

func LoadClientConfig(filename string) models.Config {
	if filename == "" {
		filename = "./config.yml"
	}
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		logrus.Fatalf("No config file found %s.", filename)
	}
	var config models.Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Printf("Invliad config file provided")
		panic(err)
	}
	return config
}
