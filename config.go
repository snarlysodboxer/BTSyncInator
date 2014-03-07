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
  configFilePath = flag.String("config", ".btsyncinator.conf", "path to config file.")
  privateKeyFilePath = flag.String("private-key", "/home/user/.ssh/id_rsa", "path to private key file.")
  config = conf.NewConfigFile()
)

func configViewHandler(writer http.ResponseWriter, request *http.Request) {
  tmpl, err := template.ParseFiles("config_view.html")
  if err != nil {
    log.Fatalf("Error with ParseFiles! %s", err)
  }
  tmpl.Execute(writer, daemons)
}

func configCreateHandler(writer http.ResponseWriter, request *http.Request) {
  // AddSection and AddOption return boolean
  if config.AddSection(request.FormValue("Name")) {
    if config.AddOption(request.FormValue("Name"), "sshUserString", request.FormValue("sshUserName")) {
      if config.AddOption(request.FormValue("Name"), "serverAddrString", request.FormValue("serverAddress")) {
        if config.AddOption(request.FormValue("Name"), "remoteAddrString", request.FormValue("remoteAddress")) {
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

