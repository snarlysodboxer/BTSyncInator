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
  "strings"
  "strconv"
)

type Daemons []Daemon

type Daemon struct {
  Name              string
  Addresses         sshPortForward.Addresses
  APIData           APIData
}

func loadDaemonsFromConfig(sections *[]string) {
  for index, section := range *sections {
    daemon := Daemon{}
    daemon.Name                           = section
    daemon.Addresses.SSHUserString, _     = config.GetString(section, "sshUserString")
    daemon.Addresses.ServerAddrString, _  = config.GetString(section, "serverAddrString")
    daemon.Addresses.RemoteAddrString, _  = config.GetString(section, "daemonAddrString")
    daemon.Addresses.LocalAddrString      = fmt.Sprintf("localhost:%d", 9000 + index)
    daemon.Addresses.PrivateKeyPathString = *privateKeyFilePath
    daemons = append(daemons, daemon)
  }
}

func (daemons *Daemons) createPortForwards() {
  for _, daemon := range *daemons {
    // Create portforward
    err := sshPortForward.ConnectAndForward(daemon.Addresses)
    if err != nil {
      log.Fatalf("Error with ConnectAndForward %v", err)
    }
  }
}

func (daemons *Daemons) loadAPIAllDatas() {
  for _, daemon := range *daemons {
    // Get port int from address string
    portStr := strings.TrimLeft(daemon.Addresses.LocalAddrString, ":")
    port, err := strconv.Atoi(portStr)
    if err != nil {
      log.Fatalf("Error with strconv.Atoi %v", err)
    }
    daemon.APIData = loadAPIAllData(port)
  }
}

type APIData struct {
  Error       error
  Folders     []Folder
  OS          *btsync.GetOSResponse
  Preferences *btsync.GetPreferencesResponse
  Speeds      *btsync.GetSpeedResponse
}

func loadAPIAllData(localPortInt int) APIData {
  api := btsync.New("", "", localPortInt, false)
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
  tmpl := template.Must(template.ParseFiles("root_view.html"))
  tmpl.Execute(writer, daemons)
}

var daemons = Daemons{}

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

  // Get Daemons from config file
  allSections := config.GetSections()
  // TODO: create a less fragile way to remove the "default" section.
  sects := allSections[1:]
  sections := &sects
  loadDaemonsFromConfig(sections)

  // Create Port Forwards
  daemons.createPortForwards()

  // Load API Datas
  daemons.loadAPIAllDatas()

  // Respond to http resquests
  http.HandleFunc("/config", configViewHandler)
  http.HandleFunc("/config/delete", configDeleteHandler)
  http.HandleFunc("/config/create", configCreateHandler)
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe("localhost:10000", nil)
}
