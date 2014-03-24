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
)

const configHeader = "Configuration file for BTSyncInator:"

var (
	configFilePath = flag.String("config", ".btsyncinator.conf", "path to config file.")
	debug          = flag.Bool("debug", false, "enable debug mode.")
	apiDebug       = flag.Bool("apiDebug", false, "enable debug mode on btsync-api library.")
	config         = conf.NewConfigFile()
	settings       = Settings{}
)

type Settings struct {
	PrivateKeyPath     string
	ServeAddress       string
  UseTLS             bool
  TLSKeyPath         string
  TLSCertPath        string
}

func readSlashCreateConfig() {
	if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(*configFilePath, 0600, configHeader)
	} else {
		config, err = conf.ReadConfigFile(*configFilePath)
		if err != nil {
			log.Fatal("Error with ReadConfigFile:", err)
		}
	}
}

// Get general settings from default section of config file
func loadSettings() {
	readSlashCreateConfig()
  // Private key file path
	privateKeyPath, err := config.GetString("default", "privateKeyPath")
	if err != nil {
    if *debug {
      log.Printf("Error with config.GetString: %s", err)
    }
	}
  if privateKeyPath == "" {
    if *debug {
      log.Println("Private key path not set, using $HOME/.ssh/id_rsa")
    }
    privateKeyPath = (os.Getenv("HOME") + "/.ssh/id_rsa")
  }
	settings.PrivateKeyPath = privateKeyPath
  // Serve Address
	serveAddress, err := config.GetString("default", "serveaddress")
	if err != nil {
    if *debug {
      log.Printf("Error with config.GetString: %s", err)
    }
	}
  if serveAddress == "" {
    if *debug {
      log.Println("Serve Address not set, using localhost:10000")
    }
    serveAddress = "localhost:10000"
  }
	settings.ServeAddress = serveAddress
  // TLS private key file path
	tlsKeyPath, err := config.GetString("default", "tlsKeyPath")
	if err != nil {
    if *debug {
      log.Printf("Error with config.GetString: %s", err)
    }
	}
	settings.TLSKeyPath = tlsKeyPath
  // TLS cert file path
	tlsCertPath, err := config.GetString("default", "tlsCertPath")
	if err != nil {
    if *debug {
      log.Printf("Error with config.GetString: %s", err)
    }
	}
	settings.TLSCertPath = tlsCertPath
  // Use TLS bool
	useTLS, err := config.GetBool("default", "useTLS")
	if err != nil {
    if *debug {
      log.Printf("Error with config.GetString: %s", err)
    }
	}
  if ! useTLS {
    if *debug {
      log.Println("Use TLS set to false.")
    }
  } else if settings.TLSKeyPath == "" && settings.TLSCertPath == "" {
    if *debug {
      log.Println("Use TLS set to true, but TLS key path and/or cert path not set, generating self-signed cert.")
    }
    genCACert("btsyncinator", 4)
    settings.TLSKeyPath = "btsyncinator.key"
    settings.TLSCertPath = "btsyncinator.crt"
  } else {
    if *debug {
      log.Println("Use TLS set to true.")
    }
  }
	settings.UseTLS = useTLS
  // Write changes to config file
  config.AddOption("default", "privateKeyPath", privateKeyPath)
  config.AddOption("default", "serveAddress", serveAddress)
  config.AddOption("default", "useTLS", fmt.Sprintf("%t", useTLS))
  config.AddOption("default", "tlsKeyPath", tlsKeyPath)
  config.AddOption("default", "tlsCertPath", tlsCertPath)
  err = config.WriteConfigFile(*configFilePath, 0600, configHeader)
  if err != nil {
    log.Fatalf("Error with WriteConfigFile: %s", err)
  }
}

func setupDaemonsFromConfig() {
	// copy old Daemons
	oldDaemons := daemons
	readSlashCreateConfig()
	allSections := config.GetSections()
	// Get Daemons from config file
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
		daemon.Addresses.PrivateKeyPathString = settings.PrivateKeyPath
		daemons = append(daemons, *daemon)
		// Setup port-forward
		if len(oldDaemons) != 0 {
			for oldIndex, _ := range oldDaemons {
				if oldDaemons[oldIndex].Name == daemon.Name {
					if oldDaemons[oldIndex].Forwarded == false {
						go sshPortForward.ConnectAndForward(daemons[sectionIndex].Addresses)
					}
					daemons[sectionIndex].Forwarded = true
				}
			}
		} else {
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
	readSlashCreateConfig()
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
	readSlashCreateConfig()
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
