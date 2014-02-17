package main

import (
  "fmt"
  btsync "github.com/vole/btsync-api"
  "log"
  "net/http"
  "html/template"
  "flag"
  "code.google.com/p/goconf/conf"
  "os"
  "github.com/snarlysodboxer/sshPortForward"
)

type DaemonAPIs []DaemonAPI

type DaemonAPI struct {
  Name              string
  ServerAddrString  string
  DaemonAddrString  string
  LocalAddrString   string
  APIData           APIData
  SSHUserString     string
}

func loadDaemonAPIs() DaemonAPIs {
  daemonAPIs := DaemonAPIs{}
  // TODO: create a less fragile way to remove the "default" section.
  sections := config.GetSections()
  for index, section := range sections[1:] {
    localPortInt := 9000 + index
    daemonAPI := DaemonAPI{}
    daemonAPI.Name                = section
    daemonAPI.ServerAddrString, _ = config.GetString(section, "serverAddrString")
    daemonAPI.DaemonAddrString, _ = config.GetString(section, "daemonAddrString")
    daemonAPI.LocalAddrString     = fmt.Sprintf("localhost:%d", localPortInt)
    daemonAPI.SSHUserString, _    = config.GetString(section, "sshUserString")
    // Create portforward
    err := sshPortForward.ConnectAndForward(daemonAPI.SSHUserString, daemonAPI.ServerAddrString, daemonAPI.LocalAddrString, daemonAPI.DaemonAddrString, *privatekeyFilePath)
    if err != nil {
      log.Fatalf("Error with ConnectAndForward %v", err)
    }
    daemonAPI.APIData = loadAPIAllData(localPortInt)
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
