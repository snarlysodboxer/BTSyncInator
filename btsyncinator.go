package main

import (
  btsync "github.com/vole/btsync-api"
  "fmt"
  "log"
  "net/http"
  "html/template"
  "flag"
  "code.google.com/p/goconf/conf"
  "os"
)

var (
  header = "Configuration file for BTSyncInator:"
  config_file_path = flag.String("config_file", ".btsyncinator.conf", "path to config file.")
  config = conf.NewConfigFile()
)

type Folder struct {
  Folder      btsync.Folder
  Files       []btsync.File
  Secrets     *btsync.GetSecretsResponse
}

type APIData struct {
  Folders     []Folder
  OS          string
  Preferences *btsync.GetPreferencesResponse
}

type DaemonAPI struct {
  Name        string
  FQDN        string
  DaemonPort  int
  LocalPort   int
  APIData     APIData
}

type DaemonAPIs []DaemonAPI

func ExtendDaemonAPIsSlice(slice []DaemonAPI, element DaemonAPI) []DaemonAPI {
    n := len(slice)
    if n == cap(slice) {
        // Slice is full; must grow.
        // We double its size and add 1, so if the size is zero we still grow.
        newSlice := make([]DaemonAPI, len(slice), 2*len(slice)+1)
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0 : n+1]
    slice[n] = element
    return slice
}

func ExtendFoldersSlice(slice []Folder, element Folder) []Folder {
    n := len(slice)
    if n == cap(slice) {
        // Slice is full; must grow.
        // We double its size and add 1, so if the size is zero we still grow.
        newSlice := make([]Folder, len(slice), 2*len(slice)+1)
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0 : n+1]
    slice[n] = element
    return slice
}

func loadAPIData(localPort int) APIData {
  api := btsync.New("", "", localPort, false)
  data := APIData{}
  // Get folders:
  folders, err := api.GetFolders()
  if err != nil {
    log.Fatalf("Error with GetFolders! %s", err)
  }
  // For each folder:
  fldrs := *new([]Folder)
  for _, folder := range *folders {
    fldr := Folder{}
    fldr.Folder = folder
    // Get Files for folder:
    files, _ := api.GetFilesForPath(folder.Secret, "")
    fldr.Files = *files
    // Get Secrets for folder:
    secrets, err := api.GetSecretsForSecret(folder.Secret)
    if err != nil {
      log.Fatalf("Error with GetSecretsForSecret! %s", err)
    }
    fldr.Secrets = secrets
    fldrs = ExtendFoldersSlice(fldrs, fldr)
  }
  data.Folders = fldrs
  // Get the OS:
  os, _ := api.GetOS()
  data.OS = os.Name
  // Get general Preferences:
  preferences, _ := api.GetPreferences()
  data.Preferences = preferences
  return data
}

func loadDaemonAPIs() DaemonAPIs {
  daemonAPIs := DaemonAPIs{}
  // TODO: create a less fragile way to remove the "default" section.
  sections := config.GetSections()
  for _, section := range sections[1:] {
    daemonAPI := DaemonAPI{}
    daemonAPI.Name = section
    daemonAPI.FQDN, _ = config.GetString(section, "fqdn")
    daemonAPI.DaemonPort, _ = config.GetInt(section, "daemon_port")
    daemonAPI.LocalPort, _ = config.GetInt(section, "local_port")
    daemonAPI.APIData = loadAPIData(daemonAPI.LocalPort)
    daemonAPIs = ExtendDaemonAPIsSlice(daemonAPIs, daemonAPI)
  }
  return daemonAPIs
}

func rootHandler(writer http.ResponseWriter, request *http.Request) {
  daemonAPIs := loadDaemonAPIs()
  tmpl := template.Must(template.ParseFiles("root_view.html"))
  tmpl.Execute(writer, daemonAPIs)
}

func configViewHandler(writer http.ResponseWriter, request *http.Request) {
  daemonAPIs := loadDaemonAPIs()
  tmpl, err := template.ParseFiles("config_view.html")
  if err != nil {
    log.Fatalf("Error with ParseFiles! %s", err)
  }
  tmpl.Execute(writer, daemonAPIs)
}

func configCreateHandler(writer http.ResponseWriter, request *http.Request) {
  // AddSection and AddOption return boolean
  if config.AddSection(request.FormValue("Name")) {
    if config.AddOption(request.FormValue("Name"), "fqdn", request.FormValue("FQDN")) {
      if config.AddOption(request.FormValue("Name"), "daemon_port", request.FormValue("Port")) {
        err := config.WriteConfigFile(*config_file_path, 0600, header)
        if err != nil {
          log.Fatalf("Error with WriteConfigFile: %s", err)
        }
      }
      daemons, err := conf.ReadConfigFile(*config_file_path)
      if err != nil {
        log.Fatalf("Error with ReadConfigFile: %s", err)
      } else {
        config = daemons
        http.Redirect(writer, request, "/config", http.StatusFound)
      }
    }
  }
}

func configDeleteHandler(writer http.ResponseWriter, request *http.Request) {
  if config.RemoveSection(request.FormValue("DeleteName")) {
    err := config.WriteConfigFile(*config_file_path, 0600, header)
    if err != nil {
      log.Fatalf("Error with WriteConfigFile: %s", err)
    }
    daemons, err := conf.ReadConfigFile(*config_file_path)
    if err != nil {
      log.Fatalf("Error with ReadConfigFile: %s", err)
    } else {
      config = daemons
      http.Redirect(writer, request, "/config", http.StatusFound)
    }
  } else {
    fmt.Fprint(writer, "Error with RemoveSection!")
  }
}

func main() {
  flag.Parse()
  if _, err := os.Stat(*config_file_path); os.IsNotExist(err) {
    config.WriteConfigFile(*config_file_path, 0600, header)
  } else {
    config, err = conf.ReadConfigFile(*config_file_path)
    if err != nil {
      log.Fatal("Error with ReadConfigFile:", err)
    }
  }

  http.HandleFunc("/config", configViewHandler)
  http.HandleFunc("/config/delete", configDeleteHandler)
  http.HandleFunc("/config/create", configCreateHandler)
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe("localhost:10000", nil)
}
//  for _, daemon := range daemons {
//    api := btsync.New("", "", daemon.Port, false)
//    fmt.Fprint(writer, heading(daemon))
//
//    // Get a list of Sync folders.
//    folders, err := api.GetFolders()
//    if err != nil {
//      log.Fatalf("Error with GetFolders! %s", err)
//    }
//    for _, folder := range *folders {
//      fmt.Fprintf(writer, "Sync folder %s has %d files\n", folder.Dir, folder.Files)
//      secrets, err := api.GetSecretsForSecret(folder.Secret)
//      if err != nil {
//        log.Fatalf("Error with GetSecretsForSecret! %s", err)
//      }
//      fmt.Fprintf(writer, "\tAnd has this read-write secret: %s\n", secrets.ReadWrite)
//      fmt.Fprintf(writer, "\tAnd has this read-only secret: %s\n", secrets.ReadOnly)
//      // Get a list of files in the folder.
//      files, err := api.GetFiles(folder.Secret)
//      if err != nil {
//        log.Fatalf("Error with GetFiles! %s", err)
//      }
//      for _, file := range *files {
//        fmt.Fprintf(writer, "\tFile %s has size %dK\n", file.Name, (file.Size/1000))
//      }
//    }
//
//    // Get Sync's current upload/download speed.
//    speed, _ := api.GetSpeed()
//    fmt.Fprintf(writer, "Speed: upload=%d, download=%d", speed.Upload, speed.Download)
//  }
