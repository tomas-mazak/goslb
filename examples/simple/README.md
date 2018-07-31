Simple GoSLB deployment with docker-compose
===========================================

This is a simple GoSLB demo, not using priorities and sites. Here, a service is created with two
HTTP endpoints. These two endpoints are returned in a DNS response for the service, in random order.
We will view the state of the service using dig (DNS) and curl (REST API). Then we make one of the
endpoints "unhealthy" and observe the behaviour.

First, we will use docker-compose to spin up 4 docker containers:

  * etcd -- backend KV store used by GoSLB
  * goslb -- the GoSLB server itself
  * web1/web2 -- endpoints we want to load-balance over


Setup
-----

  * Install [Docker](https://docs.docker.com/install/) and [docker-compose](https://docs.docker.com/compose/install/)
  * Clone this repo:
```
git clone https://github.com/tomas-mazak/goslb.git
cd goslb/examples/simple
```
  * Start the docker cotainers:
```
docker-compose up -d
```
  * Create the service:
```
WEB1_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' goslb_webserver1)
WEB2_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' goslb_webserver2)

curl -XPOST http://localhost:8080/services/ -d @- <<EOF
{
  "Domain": "web.goslb.",
  "Endpoints": [
    {
        "IP": "${WEB1_IP}", 
        "Enabled": true 
    },                
    {                  
        "IP": "${WEB2_IP}",
        "Enabled": true
    }                   
  ],                 
  "Monitor": {       
    "Type": "HTTP",
    "Interval": 10,
    "Timeout": 2,
    "Port": 8080,
    "URI": "/ready",
    "SuccessCodes": [200]
  }                      
}                      
EOF
```

Observe status
--------------

If everything went well, the previos command should have returned `{"Success": true}` json 
response. Now, we can view the status of our service's endpoints using `_cat` API:

```
curl http://localhost:8080/_cat/endpoints/web.goslb.
```

We can also resolve the service using DNS:

```
dig @localhost -p 8053 web.goslb.
```

Both endpoints should be healthy and hence returned in the DNS response (in random order).


Unhealthy endpoint
------------------

We can simulate endpoint failure by setting it's readiness state:

```
curl -XPOST ${WEB1_IP}:8080/set?ready=false
```

After this, the light webserver used for this demo will stop returning success code `200`. After
a few seconds, GoSLB monitor will notice this and mark the endpoint as unhealthy:

```
curl http://localhost:8080/_cat/endpoints/web.goslb.
```

We can verify the unhealthy endpoint was removed from DNS response:

```
dig @localhost -p 8053 web.goslb.
```
