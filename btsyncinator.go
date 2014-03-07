package main

import (
  "fmt"
  btsync "github.com/vole/btsync-api"
  "log"
  "net"
  "net/http"
  "html/template"
  "flag"
  "code.google.com/p/goconf/conf"
  "os"
  "github.com/snarlysodboxer/sshPortForward"
  "strconv"
  "time"
)

var daemons []Daemon

type Daemon struct {
  Name            string
  Addresses       sshPortForward.Addresses
  API             *btsync.BTSyncAPI
  APIData         APIData
}

type Folder struct {
  Folder      btsync.Folder
  Secrets     *btsync.GetSecretsResponse
  SyncHosts   *btsync.GetFolderHostsResponse
  Files       *btsync.GetFilesResponse
}

type APIData struct {
  Error       error
  Folders     []Folder
  OS          *btsync.GetOSResponse
  Preferences *btsync.GetPreferencesResponse
  Speeds      *btsync.GetSpeedResponse
}

func loadDaemonsFromConfig(sections *[]string) {
  for index, section := range *sections {
    daemon := &Daemon{}
    daemon.Name                           = section
    daemon.Addresses.SSHUserString, _     = config.GetString(section, "sshUserString")
    daemon.Addresses.ServerAddrString, _  = config.GetString(section, "serverAddrString")
    daemon.Addresses.RemoteAddrString, _  = config.GetString(section, "daemonAddrString")
    daemon.Addresses.LocalAddrString      = fmt.Sprintf("localhost:%d", 9000 + index)
    daemon.Addresses.PrivateKeyPathString = *privateKeyFilePath
    daemons                              = append(daemons, *daemon)
  }
}

//func setupPortForwards() {
//  for {
//    for index, _ := range daemons {
//      // Create portforward
//      log.Printf("Making Connection for %s", daemons[index].Name)
//      sshPortForward.ConnectAndForward(daemons[index].Addresses)
////      _, err := http.NewRequest("GET", daemons[index].Addresses.LocalAddrString, nil)
////      if err != nil {
////        log.Printf("Error with http.NewRequest %v", err)
////      }
//    }
//    time.Sleep(30 * time.Second)
//  }
//}

func loadAPIs() {
  for index, _ := range daemons {
    // Get port int from address string
    _, portStr, _ := net.SplitHostPort(daemons[index].Addresses.LocalAddrString)
    port, err := strconv.Atoi(portStr)
    if err != nil {
      log.Fatalf("Error with strconv.Atoi %v", err)
    }
    daemons[index].API = btsync.New("", "", port, false)
  }
}

func loadAPIAllDatas() {
  for {
    // Load APIs into each Daemon struct
    loadAPIs()
    for index, _ := range daemons {
      data := APIData{}
      data.Error = nil
      // Get the OS:
      os, err := daemons[index].API.GetOS()
      if err != nil {
        data.Error = err
      }
      data.OS = os
      // Get Upload and Download speed
      speeds, err := daemons[index].API.GetSpeed()
      if err != nil {
        data.Error = err
      }
      data.Speeds = speeds
      // Get general Preferences:
      preferences, err := daemons[index].API.GetPreferences()
      if err != nil {
        // TODO: Fix "json: cannot unmarshal number into Go value of type bool" bug
        //data.Error = err
        //fmt.Printf("Error with GetPreferences! %s", err)
      }
      data.Preferences = preferences
      daemons[index].APIData = data
      // Get All Folders data
      if daemons[index].APIData.Error != nil {
        log.Printf("Error loading APIData for %s: %v", daemons[index].Name, daemons[index].APIData.Error)
      }
    }
    loadAPIFoldersDatas()
    time.Sleep(30 * time.Second)
  }
}

func loadAPIFoldersDatas() {
  for index, _ := range daemons {
    apiFolders, err := daemons[index].API.GetFolders()
    folders := []Folder{}
    for _, apiFolder := range *apiFolders {
      folder := &Folder{}
      folder.Folder = apiFolder
      // Get Files for folder:
      folder.Files, err = daemons[index].API.GetFilesForPath(apiFolder.Secret, "")
      // Get Secrets for folder:
      folder.Secrets, err = daemons[index].API.GetSecretsForSecret(apiFolder.Secret)
      // Get Known Hosts for folder:
      //// TODO: Fix "json: cannot unmarshal object into Go
      ////     value of type btsync_api.GetFolderHostsResponse" bug
      ////folder.SyncHosts, err = daemons[index].API.GetFolderHosts(apiFolder.Secret)
      folder.SyncHosts, _ = daemons[index].API.GetFolderHosts(apiFolder.Secret)

      folders = append(folders, *folder)
    }
    daemons[index].APIData.Error    = err
    daemons[index].APIData.Folders  = folders
  }
}

func rootHandler(writer http.ResponseWriter, request *http.Request) {
  tmpl := template.Must(template.ParseFiles("root_view.html"))
  tmpl.Execute(writer, daemons)
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

  // Get Daemons from config file
  allSections := config.GetSections()
  // TODO: create a less fragile way to remove the "default" section.
  sects := allSections[1:]
  sections := &sects
  loadDaemonsFromConfig(sections)

  // Create Port Forwards
  // TODO: create quitChan
  //quitChan, err := daemons.setupPortForwards()
  //if err != nil {
  //  log.Fatalf("Error with daemons.setupPortForwards: %v", err)
  //}
//  go setupPortForwards()
//  time.Sleep(10 * time.Second)

  // Load API Datas
  go loadAPIAllDatas()

  // Respond to http resquests
  http.HandleFunc("/config", configViewHandler)
  http.HandleFunc("/config/delete", configDeleteHandler)
  http.HandleFunc("/config/create", configCreateHandler)
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe("localhost:10000", nil)
}
