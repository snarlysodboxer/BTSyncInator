package main

import (
	"code.google.com/p/goconf/conf"
	"flag"
	"fmt"
	"github.com/snarlysodboxer/sshPortForward"
	"html/template"
	"log"
	"net/http"
	"os"
	//"time"
)

const configHeader = "Configuration file for BTSyncInator:"

var (
	configFilePath     = flag.String("config", ".btsyncinator.conf", "path to config file.")
	privateKeyFilePath = flag.String("private-key", "/home/user/.ssh/id_rsa", "path to private key file.")
	debug              = flag.Bool("debug", false, "enable debug mode.")
	apiDebug           = flag.Bool("apiDebug", false, "enable debug mode on btsync-api library.")
	config             = conf.NewConfigFile()
)

func readSlashCreateConfig() *conf.ConfigFile {
	if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(*configFilePath, 0600, configHeader)
	} else {
		config, err = conf.ReadConfigFile(*configFilePath)
		if err != nil {
			log.Fatal("Error with ReadConfigFile:", err)
		}
	}
	return config
}

func setupDaemonsFromConfig() {
	// copy old Daemons
	oldDaemons := daemons
	if *debug {
		log.Printf("oldDaemons: %v", oldDaemons)
	}
	// Get Daemons from config file
	config := readSlashCreateConfig()
	allSections := config.GetSections()
	sects := allSections[1:]
	sections := &sects
	daemons = []Daemon{}
	for sectionIndex, section := range *sections {
		daemon := &Daemon{}
		daemon.Name = section
		daemon.Addresses.SSHUserString, _ = config.GetString(section, "sshUserString")
		daemon.Addresses.ServerAddrString, _ = config.GetString(section, "serverAddrString")
		daemon.Addresses.RemoteAddrString, _ = config.GetString(section, "daemonAddrString")
		daemon.Addresses.LocalAddrString = fmt.Sprintf("localhost:%d", 9000+sectionIndex)
		daemon.Addresses.PrivateKeyPathString = *privateKeyFilePath
		daemons = append(daemons, *daemon)
		// Setup portforward
		if len(oldDaemons) != 0 {
			for oldIndex, _ := range oldDaemons {
				if *debug {
					log.Printf("oldDaemons: %v", oldDaemons[oldIndex].Addresses)
				}
				if oldDaemons[oldIndex].Name == daemon.Name {
					if oldDaemons[oldIndex].Forwarded == false {
						go sshPortForward.ConnectAndForward(daemons[sectionIndex].Addresses)
					}
					daemons[sectionIndex].Forwarded = true
				}
			}
		} else {
			if *debug {
				log.Println(sectionIndex, daemons[sectionIndex])
			}
			go sshPortForward.ConnectAndForward(daemons[sectionIndex].Addresses)
			daemons[sectionIndex].Forwarded = true
		}
	}
}

func configViewHandler(writer http.ResponseWriter, request *http.Request) {
	tmpl, err := template.ParseFiles("config_view.html")
	if err != nil {
		log.Fatalf("Error with ParseFiles! %s", err)
	}
	tmpl.Execute(writer, daemons)
}

func configCreateHandler(writer http.ResponseWriter, request *http.Request) {
	config := readSlashCreateConfig()
	// AddSection and AddOption return boolean
	if config.AddSection(request.FormValue("Name")) {
		if config.AddOption(request.FormValue("Name"), "sshUserString", request.FormValue("sshUserName")) {
			if config.AddOption(request.FormValue("Name"), "serverAddrString", request.FormValue("serverAddress")) {
				if config.AddOption(request.FormValue("Name"), "daemonAddrString", request.FormValue("daemonAddress")) {
					err := config.WriteConfigFile(*configFilePath, 0600, configHeader)
					if err != nil {
						log.Fatalf("Error with WriteConfigFile: %s", err)
					}
				}
				dmns, err := conf.ReadConfigFile(*configFilePath)
				if err != nil {
					log.Fatalf("Error with ReadConfigFile: %s", err)
				} else {
					config = dmns
					setupDaemonsFromConfig()
					http.Redirect(writer, request, "/config", http.StatusFound)
				}
			}
		}
	}
}

func configDeleteHandler(writer http.ResponseWriter, request *http.Request) {
	config := readSlashCreateConfig()
	if config.RemoveSection(request.FormValue("DeleteName")) {
		err := config.WriteConfigFile(*configFilePath, 0600, configHeader)
		if err != nil {
			log.Fatalf("Error with WriteConfigFile: %s", err)
		}
		dmns, err := conf.ReadConfigFile(*configFilePath)
		if err != nil {
			log.Fatalf("Error with ReadConfigFile: %s", err)
		} else {
			config = dmns
			setupDaemonsFromConfig()
			http.Redirect(writer, request, "/config", http.StatusFound)
		}
	} else {
		log.Fatal(writer, "Error with RemoveSection!")
	}
}
