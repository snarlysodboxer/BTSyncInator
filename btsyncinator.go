package main

import (
	"flag"
	"github.com/snarlysodboxer/sshPortForward"
	btsync "github.com/vole/btsync-api"
	"html/template"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

var (
	daemons []Daemon
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

func loadAPIs() {
	for index, _ := range daemons {
		// Get port int from address string
		_, portStr, _ := net.SplitHostPort(daemons[index].Addresses.LocalAddrString)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("Error with strconv.Atoi %v", err)
		}
		daemons[index].API = btsync.New("", "", port, *apiDebug)
	}
}

func loadAPIAllDatas() {
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
	loadAPIAllDatas()
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

	loadSettings()

	setupDaemonsFromConfig()

	// Respond to http resquests
	http.HandleFunc("/config", configViewHandler)
	http.HandleFunc("/config/delete", configDeleteHandler)
	http.HandleFunc("/config/create", configCreateHandler)
	http.HandleFunc("/folder/add/new", folderAddNewHandler)
	http.HandleFunc("/folder/add/existing", folderAddExistingHandler)
	http.HandleFunc("/folder/remove", folderRemoveHandler)
	http.HandleFunc("/", rootHandler)
  if settings.UseTLS {
    http.ListenAndServeTLS(settings.ServeAddress, settings.TLSCertPath, settings.TLSKeyPath, nil)
  } else {
    http.ListenAndServe(settings.ServeAddress, nil)
  }
}
