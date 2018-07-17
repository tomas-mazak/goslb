package goslb

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"net"
)

type ApiStatus struct {
	 Success bool
	 Msg string `json:",omitempty"`
}

func getServices(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(serviceDomain.services)
}

func getService(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if service, found := serviceDomain.services[name]; found {
		json.NewEncoder(w).Encode(service)
	} else {
		http.Error(w, "", http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found"})
	}
}

func addService(w http.ResponseWriter, r *http.Request) {
	service := &Service{}

	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, "", http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	if serviceDomain.Exists(service.Domain) {
		return
	}

	if err := serviceDomain.Add(service); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
	}

	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func deleteService(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if ! serviceDomain.Exists(name) {
		http.Error(w, "", http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found"})
		return
	}

	if err := serviceDomain.Delete(name); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
	}

	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func getEndpoints(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["servicename"]
	ipstr, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(ipstr)

	if ! serviceDomain.Exists(serviceName) {
		http.Error(w, "", http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found"})
		return
	}

	json.NewEncoder(w).Encode(serviceDomain.Get(serviceName).GetOrdered(ip))
}

type CatAPI struct {}

func (api *CatAPI) getEndpoints(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["servicename"]

	if ! serviceDomain.Exists(serviceName) {
		http.Error(w, "", http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("Service not found: %v\n", serviceName)))
		return
	}

	tab := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	fmt.Fprint(tab, "IP\tEnabled\tHealthy\tPriority\tSite\n")
	for _, ep := range serviceDomain.Get(serviceName).Endpoints {
		fmt.Fprintf(tab, "%v\t%v\t%v\t%v\t%v\n", ep.IP, ep.Enabled, ep.Healthy, ep.Priority, ep.Site)
	}
	tab.Flush()
}

func (api *CatAPI) getClientSite(w http.ResponseWriter, r *http.Request) {
	ipstr, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(ipstr)
	if site, found := siteMatcher.GetSite(ip); found {
		w.Write([]byte(fmt.Sprintf("%v: %v\n", ip, site)))
	} else {
		w.Write([]byte(fmt.Sprintf("%v: SITE UNDEFINED\n", ip)))
	}
}

func InitApiServer(config *Config) {
	catApi := &CatAPI{}

    router := mux.NewRouter()
	router.HandleFunc("/services", getServices).Methods("GET")
	router.HandleFunc("/services", addService).Methods("POST")
	router.HandleFunc("/services/{name}", getService).Methods("GET")
	//router.HandleFunc("/services/{name}", applyService).Methods("PUT")
	router.HandleFunc("/services/{name}", deleteService).Methods("DELETE")
    router.HandleFunc("/endpoints/{servicename}", getEndpoints).Methods("GET")
	router.HandleFunc("/_cat/endpoints/{servicename}", catApi.getEndpoints).Methods("GET")
	router.HandleFunc("/_cat/clientsite", catApi.getClientSite).Methods("GET")
    log.Infof("Starting API on %v", config.BindAddrAPI)

    if err := http.ListenAndServe(config.BindAddrAPI, router); err != nil {
		log.WithError(err).Fatal("Failed to start API")
	}
}


