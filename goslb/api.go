package goslb

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"net"
	"github.com/gorilla/handlers"
	"os"
	"strings"
	"sort"
)

type ApiStatus struct {
	 Success bool
	 Msg string `json:",omitempty"`
}

type ServiceAPI struct {}

func (api *ServiceAPI) list(w http.ResponseWriter, r *http.Request) {
	ret := make([]string, len(serviceDomain.services))[:0]
	for k := range serviceDomain.services {
		ret = append(ret, k)
	}
	json.NewEncoder(w).Encode(ret)
}

func (api *ServiceAPI) get(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if service, found := serviceDomain.services[name]; found {
		json.NewEncoder(w).Encode(service)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found: " + name})
	}
}

func (api *ServiceAPI) set(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	service := &Service{}
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	if service.Domain != "" && service.Domain != name {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "URI and JSON names don't match"})
		return
	}
	service.Domain = name

	if err := serviceDomain.IsValid(service); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	if !serviceDomain.Exists(service.Domain) {
		if err := serviceDomain.Add(service); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
			return
		}
	} else {
		if err := serviceDomain.Update(service); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
			return
		}
	}

	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func (api *ServiceAPI) add(w http.ResponseWriter, r *http.Request) {
	service := &Service{}

	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "JSON parse error: " + err.Error()})
		return
	}

	if serviceDomain.Exists(service.Domain) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service already exists: " + service.Domain})
		return
	}
	if err := serviceDomain.IsValid(service); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	if err := serviceDomain.Add(service); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
		return
	}

	w.Header().Set("Location", "/services/" + service.Domain)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func (api *ServiceAPI) delete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if ! serviceDomain.Exists(name) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found: " + name})
		return
	}

	if err := serviceDomain.Delete(name); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: err.Error()})
	}

	json.NewEncoder(w).Encode(ApiStatus{Success: true})
}

func (api *ServiceAPI) dnsResponse(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	ipstr, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(ipstr)

	if ! serviceDomain.Exists(name) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ApiStatus{Success: false, Msg: "Service not found: " + name})
		return
	}

	json.NewEncoder(w).Encode(serviceDomain.Get(name).GetOrdered(ip))
}

type CatAPI struct {}

func (api *CatAPI) getStatus(w http.ResponseWriter, r *http.Request) {
	tab := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprint(tab, "Node\tServices\tActiveMonitors\n")
	fmt.Fprintf(tab, "%v\t%v\t%v\n", serverStatus.nodeName, serviceDomain.Count(), serverStatus.monitors)
	tab.Flush()
}

func (api *CatAPI) getServices(w http.ResponseWriter, r *http.Request) {
	tab := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprint(tab, "Domain\tEndpoints\tHealthy\tList\n")
	svcList := serviceDomain.List()
	sort.Slice(svcList, func(i, j int) bool {
		return svcList[i].Domain < svcList[j].Domain
	})
	for _, svc := range svcList {
		var healthy []string
		for _, ep := range svc.Endpoints {
			if ep.Enabled && ep.Healthy {
				healthy = append(healthy, fmt.Sprintf("%v (%v)", ep.IP, ep.Site))
			}
		}
		fmt.Fprintf(tab, "%v\t%v\t%v\t%v\n", svc.Domain, len(svc.Endpoints), len(healthy), strings.Join(healthy, ", "))
	}
	tab.Flush()
}

func (api *CatAPI) getEndpoints(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["servicename"]

	if ! serviceDomain.Exists(serviceName) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("Service not found: %v\n", serviceName)))
		return
	}

	tab := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprint(tab, "IP\tEnabled\tHealthy\tPriority\tSite\tError\n")
	for _, ep := range serviceDomain.Get(serviceName).Endpoints {
		var errStr string
		if ep.lastError != nil {
			errStr = ep.lastError.Error()
		}
		fmt.Fprintf(tab, "%v\t%v\t%v\t%v\t%v\t%v\n", ep.IP, ep.Enabled, ep.Healthy, ep.Priority, ep.Site, errStr)
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
    router := mux.NewRouter()
	router.StrictSlash(true) // this actually means endpoints both with and without trailing slash will work

    serviceRouter := router.PathPrefix("/services").Subrouter()
    serviceApi := &ServiceAPI{}
	serviceRouter.HandleFunc("/", serviceApi.list).Methods("GET")
	serviceRouter.HandleFunc("/", serviceApi.add).Methods("POST")
	serviceRouter.HandleFunc("/{name}", serviceApi.get).Methods("GET")
	serviceRouter.HandleFunc("/{name}", serviceApi.set).Methods("PUT")
	serviceRouter.HandleFunc("/{name}", serviceApi.delete).Methods("DELETE")
    serviceRouter.HandleFunc("/{name}/dnsresponse", serviceApi.dnsResponse).Methods("GET")

	catRouter := router.PathPrefix("/_cat").Subrouter()
	catApi := &CatAPI{}
	catRouter.HandleFunc("/status", catApi.getStatus).Methods("GET")
	catRouter.HandleFunc("/services", catApi.getServices).Methods("GET")
	catRouter.HandleFunc("/endpoints/{servicename}", catApi.getEndpoints).Methods("GET")
	catRouter.HandleFunc("/clientsite", catApi.getClientSite).Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

    log.Infof("Starting API on %v", config.BindAddrAPI)
    if err := http.ListenAndServe(config.BindAddrAPI, loggedRouter); err != nil {
		log.WithError(err).Fatal("Failed to start API")
	}
}
