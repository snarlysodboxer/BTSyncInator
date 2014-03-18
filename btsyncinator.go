package main

import (
	"code.google.com/p/goconf/conf"
	"flag"
	"fmt"
	"github.com/snarlysodboxer/sshPortForward"
	btsync "github.com/vole/btsync-api"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	daemons             []Daemon
	loadAPIAllDatasChan = make(chan bool, 1)
)

type Daemon struct {
	// Using Name as unique identifier
	// TODO: enforce unique Daemon Names in config.
	Name      string
	Addresses sshPortForward.Addresses
	API       *btsync.BTSyncAPI
	APIData   APIData
	Forwarded bool
}

type Folder struct {
	Folder    btsync.Folder
	Secrets   *btsync.GetSecretsResponse
	SyncHosts *btsync.GetFolderHostsResponse
	Files     *btsync.GetFilesResponse
}

type APIData struct {
	Error       error
	Folders     []Folder
	OS          *btsync.GetOSResponse
	Preferences *btsync.GetPreferencesResponse
	Speeds      *btsync.GetSpeedResponse
	ReadTime    string
}

func loadDaemonsFromConfig() {
	// Get Daemons from config file
	allSections := config.GetSections()
	// TODO: create a less fragile way to remove the "default" section.
	sects := allSections[1:]
	sections := &sects
	for index, section := range *sections {
		daemon := &Daemon{}
		daemon.Name = section
		daemon.Forwarded = false
		daemon.Addresses.SSHUserString, _ = config.GetString(section, "sshUserString")
		daemon.Addresses.ServerAddrString, _ = config.GetString(section, "serverAddrString")
		daemon.Addresses.RemoteAddrString, _ = config.GetString(section, "daemonAddrString")
		daemon.Addresses.LocalAddrString = fmt.Sprintf("localhost:%d", 9000+index)
		daemon.Addresses.PrivateKeyPathString = *privateKeyFilePath
		daemons = append(daemons, *daemon)
	}
}

func setupPortForwards() {
	for {
		for index, _ := range daemons {
			if daemons[index].Forwarded == false {
				if *debug {
					log.Printf("daemon addresses: %s", daemons[index].Addresses)
				}
				// Create portforward
				go sshPortForward.ConnectAndForward(daemons[index].Addresses)
				daemons[index].Forwarded = true
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func loadAPIs() {
	for index, _ := range daemons {
		// Get port int from address string
		_, portStr, _ := net.SplitHostPort(daemons[index].Addresses.LocalAddrString)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("Error with strconv.Atoi %v", err)
		}
		daemons[index].API = btsync.New("", "", port, *debug)
	}
}

func loadAPIAllDatasEveryXSeconds(seconds time.Duration) {
	for {
		loadAPIAllDatasChan <- true
		time.Sleep(seconds)
	}
}

func loadAPIAllDatas() {
	for {
		select {
		case <-loadAPIAllDatasChan:
			// Load APIs into each Daemon struct
			loadAPIs()
			for index, _ := range daemons {
				data := APIData{}
				data.Error = nil

				// Get the read time:
				data.ReadTime = time.Now().String()

				// Get the OS:
				data.OS, data.Error = daemons[index].API.GetOS()
				if *debug && data.Error != nil {
					log.Printf("Error: %v", data.Error)
				}

				// Get Upload and Download speed
				data.Speeds, data.Error = daemons[index].API.GetSpeed()
				if *debug && data.Error != nil {
					log.Printf("Error: %v", data.Error)
				}

				// Get general Preferences:
				// TODO: Fix "json: cannot unmarshal number into Go value of type bool" bug
				data.Preferences, _ = daemons[index].API.GetPreferences()
				//data.Preferences, data.Error := daemons[index].API.GetPreferences()
				//if *debug && data.Error != nil { log.Printf("Error: %v", data.Error) }

				daemons[index].APIData = data
			}
			// Get All Folders data
			loadAPIFoldersDatas()
			//if *debug { log.Printf("the daemons: %v", daemons) }
		default:
		}
	}
}

func loadAPIFoldersDatas() {
	for index, _ := range daemons {
		apiFolders, err := daemons[index].API.GetFolders()
		if *debug && err != nil {
			log.Printf("Error: %v", err)
		}
		folders := []Folder{}
		for _, apiFolder := range *apiFolders {
			folder := &Folder{}
			folder.Folder = apiFolder
			// Get Files for folder:
			folder.Files, err = daemons[index].API.GetFilesForPath(apiFolder.Secret, "")
			if *debug && err != nil {
				log.Printf("Error: %v", err)
			}
			// Get Secrets for folder:
			folder.Secrets, err = daemons[index].API.GetSecretsForSecret(apiFolder.Secret)
			if *debug && err != nil {
				log.Printf("Error: %v", err)
			}
			// Get Known Hosts for folder:
			//// TODO: Fix "json: cannot unmarshal object into Go
			////     value of type btsync_api.GetFolderHostsResponse" bug
			////folder.SyncHosts, err = daemons[index].API.GetFolderHosts(apiFolder.Secret)
			folder.SyncHosts, _ = daemons[index].API.GetFolderHosts(apiFolder.Secret)
			if *debug && err != nil {
				log.Printf("Error: %v", err)
			}

			folders = append(folders, *folder)
		}
		daemons[index].APIData.Error = err
		daemons[index].APIData.Folders = folders
	}
}

func rootHandler(writer http.ResponseWriter, request *http.Request) {
	tmpl := template.Must(template.ParseFiles("root_view.html"))
	tmpl.Execute(writer, daemons)
}

func folderAddNewHandler(writer http.ResponseWriter, request *http.Request) {
	for index, _ := range daemons {
		if request.FormValue("DaemonName") == daemons[index].Name {
			_, err := daemons[index].API.AddFolder(request.FormValue("FullPath"))
			if err != nil {
				if *debug {
					log.Printf("Error with API.AddFolder: %v", err)
				}
			} else {
				time.Sleep(3 * time.Second)
				loadAPIAllDatasChan <- true
				http.Redirect(writer, request, "/", http.StatusFound)
			}
		}
	}
}

func folderAddExistingHandler(writer http.ResponseWriter, request *http.Request) {
	for index, _ := range daemons {
		if request.FormValue("DaemonName") == daemons[index].Name {
			_, err := daemons[index].API.AddFolderWithSecret(request.FormValue("FullPath"), request.FormValue("Secret"))
			if err != nil {
				if *debug {
					log.Printf("Error with API.AddFolderWithSecret: %v", err)
				}
			} else {
				time.Sleep(3 * time.Second)
				loadAPIAllDatasChan <- true
				http.Redirect(writer, request, "/", http.StatusFound)
			}
		}
	}
}

func folderRemoveHandler(writer http.ResponseWriter, request *http.Request) {
	for index, _ := range daemons {
		if request.FormValue("DaemonName") == daemons[index].Name {
			_, err := daemons[index].API.RemoveFolder(request.FormValue("RemoveSecret"))
			if err != nil {
				if *debug {
					log.Printf("Error: %v", err)
				}
			} else {
				time.Sleep(3 * time.Second)
				loadAPIAllDatasChan <- true
				http.Redirect(writer, request, "/", http.StatusFound)
			}
		}
	}
}

func main() {
	// Parse Command line flags
	flag.Parse()
	if *debug {
		log.Println("Debug mode enabled")
	}

	// Load or create config file
	if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(*configFilePath, 0600, configHeader)
	} else {
		config, err = conf.ReadConfigFile(*configFilePath)
		if err != nil {
			log.Fatal("Error with ReadConfigFile:", err)
		}
	}

	loadDaemonsFromConfig()

	// Create Port Forwards
	// TODO: create quitChan
	go setupPortForwards()
	time.Sleep(3 * time.Second)

	// Load API Datas
	go loadAPIAllDatas()
	seconds, err := time.ParseDuration("30s")
	if *debug {
		log.Printf("ParseDuration: %d", seconds)
	}
	if err != nil {
		if *debug {
			log.Printf("Error with time.ParseDuration: %v", err)
		}
	}
	go loadAPIAllDatasEveryXSeconds(seconds)

	// Respond to http resquests
	http.HandleFunc("/config", configViewHandler)
	http.HandleFunc("/config/delete", configDeleteHandler)
	http.HandleFunc("/config/create", configCreateHandler)
	http.HandleFunc("/folder/add/new", folderAddNewHandler)
	http.HandleFunc("/folder/add/existing", folderAddExistingHandler)
	http.HandleFunc("/folder/remove", folderRemoveHandler)
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe("localhost:10000", nil)
}
