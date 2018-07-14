package goslb

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
)

type ApiStatus struct {
	 Success bool
	 Msg string `json:",omitempty"`
}


func ApiServer(config *Config) {

    router := mux.NewRouter()
	router.HandleFunc("/services", getServices).Methods("GET")
	router.HandleFunc("/services", addService).Methods("POST")
	router.HandleFunc("/services/{name}", getService).Methods("GET")
	//router.HandleFunc("/services/{name}", applyService).Methods("PUT")
	router.HandleFunc("/services/{name}", deleteService).Methods("DELETE")
    Log.Infof("Starting API on %v", config.BindAddrAPI)

    if err := http.ListenAndServe(config.BindAddrAPI, router); err != nil {
		Log.WithError(err).Fatal("Failed to start API")
	}
}

func getServices(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Services)
}

func getService(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if service, found := Services[name]; found {
		json.NewEncoder(w).Encode(service)
	} else {
		http.Error(w, "", http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found"})
	}
}

func addService(w http.ResponseWriter, r *http.Request) {
	var service Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, "", http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	if _, found := Services[service.Domain]; found {
		return
	}

	Services[service.Domain] = service
	for i := range service.Endpoints {
		service.Endpoints[i].MonitorInstance = &TcpMonitor{ monitor: &service.Monitor, endpoint: &service.Endpoints[i]}
		go service.Endpoints[i].MonitorInstance.start()
	}
	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func deleteService(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	service, found := Services[name]
	if !found {
		http.Error(w, "", http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found"})
		return
	}
	for _, ep := range service.Endpoints {
		ep.MonitorInstance.stop()
	}
	delete(Services, name)
	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}
