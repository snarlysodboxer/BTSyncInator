package main

import (
	"code.google.com/p/goconf/conf"
	"flag"
	"html/template"
	"log"
	"net/http"
)

const configHeader = "Configuration file for BTSyncInator:"

var (
	configFilePath     = flag.String("config", ".btsyncinator.conf", "path to config file.")
	privateKeyFilePath = flag.String("private-key", "/home/user/.ssh/id_rsa", "path to private key file.")
	debug              = flag.Bool("debug", false, "enable debug mode.")
	config             = conf.NewConfigFile()
)

func configViewHandler(writer http.ResponseWriter, request *http.Request) {
	daemons = []Daemon{}
	loadDaemonsFromConfig()
	tmpl, err := template.ParseFiles("config_view.html")
	if err != nil {
		log.Fatalf("Error with ParseFiles! %s", err)
	}
	tmpl.Execute(writer, daemons)
}

func configCreateHandler(writer http.ResponseWriter, request *http.Request) {
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
					http.Redirect(writer, request, "/config", http.StatusFound)
				}
			}
		}
	}
}

func configDeleteHandler(writer http.ResponseWriter, request *http.Request) {
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
			http.Redirect(writer, request, "/config", http.StatusFound)
		}
	} else {
		log.Fatal(writer, "Error with RemoveSection!")
	}
}
