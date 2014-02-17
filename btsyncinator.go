package main

import (
  //"fmt"
  btsync "github.com/vole/btsync-api"
  "log"
  "net/http"
  "html/template"
  "flag"
  "code.google.com/p/goconf/conf"
  "os"
  //"github.com/snarlysodboxer/portforward"
)

const configHeader = "Configuration file for BTSyncInator:"
var (
  configFilePath = flag.String("config_file", ".btsyncinator.conf", "path to config file.")
  //configFilePath = flag.String("privatekey_file", "$HOME/.ssh/id_rsa", "path to privatekey file.")
  config = conf.NewConfigFile()
)

type DaemonAPIs []DaemonAPI

type DaemonAPI struct {
  Name        string
  FQDN        string
  DaemonPort  int
  LocalPort   int
  APIData     APIData
}

func loadDaemonAPIs() DaemonAPIs {
  daemonAPIs := DaemonAPIs{}
  // TODO: create a less fragile way to remove the "default" section.
  sections := config.GetSections()
  for index, section := range sections[1:] {
    daemonAPI := DaemonAPI{}
    daemonAPI.Name = section
    daemonAPI.FQDN, _ = config.GetString(section, "fqdn")
    daemonAPI.DaemonPort, _ = config.GetInt(section, "daemon_port")
    daemonAPI.LocalPort = 9000 + index
    daemonAPI.APIData = loadAPIAllData(daemonAPI.LocalPort)
    daemonAPIs = append(daemonAPIs, daemonAPI)
  }
  return daemonAPIs
}

type APIData struct {
  Error       error
  Folders     []Folder
  OS          *btsync.GetOSResponse
  Preferences *btsync.GetPreferencesResponse
  Speeds      *btsync.GetSpeedResponse
}

func loadAPIAllData(localPort int) APIData {
  api := btsync.New("", "", localPort, false)
  data := APIData{}
  data.Error = nil
  folders, err := loadAPIFoldersData(api)
  if err != nil {
    data.Error = err
  }
  data.Folders = folders
  // Get the OS:
  os, err := api.GetOS()
  if err != nil {
    data.Error = err
  }
  data.OS = os
  // Get Upload and Download speed
  speeds, err := api.GetSpeed()
  if err != nil {
    data.Error = err
  }
  data.Speeds = speeds
  // Get general Preferences:
  preferences, err := api.GetPreferences()
  if err != nil {
    // TODO: Fix "json: cannot unmarshal number into Go value of type bool" bug
    //data.Error = err
    //fmt.Printf("Error with GetPreferences! %s", err)
  }
  data.Preferences = preferences
  return data
}

type Folder struct {
  Folder      btsync.Folder
  Secrets     *btsync.GetSecretsResponse
  SyncHosts   *btsync.GetFolderHostsResponse
  Files       *btsync.GetFilesResponse
}

func loadAPIFoldersData(api *btsync.BTSyncAPI) ([]Folder, error) {
  var err error = nil
  folders, err := api.GetFolders()
  fldrs := *new([]Folder)
  for _, folder := range *folders {
    fldr := &Folder{}
    fldr.Folder = folder
    // Get Files for folder:
    fldr.Files, err = api.GetFilesForPath(folder.Secret, "")
    // Get Secrets for folder:
    fldr.Secrets, err = api.GetSecretsForSecret(folder.Secret)
    // Get Known Hosts for folder:
    //// TODO: Fix "json: cannot unmarshal object into Go
    ////     value of type btsync_api.GetFolderHostsResponse" bug
    ////fldr.SyncHosts, err = api.GetFolderHosts(folder.Secret)
    fldr.SyncHosts, _ = api.GetFolderHosts(folder.Secret)

    fldrs = append(fldrs, *fldr)
  }
  return fldrs, err
}

func rootHandler(writer http.ResponseWriter, request *http.Request) {
  daemonAPIs := loadDaemonAPIs()
  tmpl := template.Must(template.ParseFiles("root_view.html"))
  tmpl.Execute(writer, daemonAPIs)
}

func main() {
  // Parse Command line flags
  flag.Parse()

  // Load or create config file
  if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
    config.WriteConfigFile(*configFilePath, 0600, configHeader)
  } else {
    config, err = conf.ReadConfigFile(*configFilePath)
    if err != nil {
      log.Fatal("Error with ReadConfigFile:", err)
    }
  }

  // Respond to http resquests
  http.HandleFunc("/config", configViewHandler)
  http.HandleFunc("/config/delete", configDeleteHandler)
  http.HandleFunc("/config/create", configCreateHandler)
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe("localhost:10000", nil)
}
