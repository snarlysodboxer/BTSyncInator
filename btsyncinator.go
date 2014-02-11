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
  //"github.com/snarlysodboxer/btsyncinator/portforward"
)

var (
  header = "Configuration file for BTSyncInator:"
  config_file_path = flag.String("config_file", ".btsyncinator.conf", "path to config file.")
  config = conf.NewConfigFile()
)

type Folder struct {
  Folder      btsync.Folder
  Files       *btsync.GetFilesResponse
  Secrets     *btsync.GetSecretsResponse
}

type DaemonAPI struct {
  Name        string
  FQDN        string
  DaemonPort  int
  LocalPort   int
  APIData     APIData
}

type APIData struct {
  Error       error
  Folders     []Folder
  OS          string
  Preferences *btsync.GetPreferencesResponse
  Speeds      *btsync.GetSpeedResponse
}

type DaemonAPIs []DaemonAPI

func loadAPIData(localPort int) APIData {
  api := btsync.New("", "", localPort, false)
  data := APIData{}
  data.Error = nil
  folders, err := loadFolders(api)
  if err != nil {
    data.Error = err
  }
  data.Folders = folders
  // Get the OS:
  os, err := api.GetOS()
  if err != nil {
    data.Error = err
  }
  data.OS = os.Name
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

func loadFolders(api *btsync.BTSyncAPI) ([]Folder, error) {
  var heir error = nil
  folders, err := api.GetFolders()
  if err != nil {
    heir = err
  }
  fldrs := *new([]Folder)
  for _, folder := range *folders {
    fldr := &Folder{}
    fldr.Folder = folder
    // Get Files for folder:
    files, err := api.GetFilesForPath(folder.Secret, "")
    if err != nil {
      heir = err
    }
    fldr.Files = files
    // Get Secrets for folder:
    secrets, err := api.GetSecretsForSecret(folder.Secret)
    if err != nil {
      heir = err
    }
    fldr.Secrets = secrets
    fldrs = append(fldrs, *fldr)
  }
  return fldrs, heir
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
    daemonAPI.APIData = loadAPIData(daemonAPI.LocalPort)
    daemonAPIs = append(daemonAPIs, daemonAPI)
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
    log.Fatal(writer, "Error with RemoveSection!")
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
