package main

import (
  "log"
  "net/http"
  "html/template"
  "code.google.com/p/goconf/conf"
  "flag"
)

const configHeader = "Configuration file for BTSyncInator:"

var (
  configFilePath = flag.String("config_file", ".btsyncinator.conf", "path to config file.")
  privatekeyFilePath = flag.String("privatekey_file", "/home/user/.ssh/id_rsa", "path to privatekey file.")
  config = conf.NewConfigFile()
)

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
        err := config.WriteConfigFile(*configFilePath, 0600, configHeader)
        if err != nil {
          log.Fatalf("Error with WriteConfigFile: %s", err)
        }
      }
      daemons, err := conf.ReadConfigFile(*configFilePath)
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
    err := config.WriteConfigFile(*configFilePath, 0600, configHeader)
    if err != nil {
      log.Fatalf("Error with WriteConfigFile: %s", err)
    }
    daemons, err := conf.ReadConfigFile(*configFilePath)
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

