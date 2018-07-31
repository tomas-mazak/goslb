GoSLB - DNS load balancer
=========================

**!! This is a work-in-progress repo !!**

GoSLB is multi-site DNS load-balancer with a RESTful API and etcd backend. The basic configuration
unit is a _service_ (DNS domain) with a list of _endpoints_ (IP addresses) and a TCP or a HTTP 
health _monitor_. Each endpoint can be assigned to a _site_ (datacenter, cloud region, ...) and,
based on provided site's IP ranges, local-site endpoints are preferred in the DNS responses to 
clients.


Quick start
-----------

To get a demo instance running quickly, have a look at the [simple docker example](examples/simple/).


Endpoint selection
------------------

Algorithm that chooses the list of endpoints to return to the client is based on enabled/disabled
state, priority, health (as reported by monitor) and site affiliation:

1. Only enabled healthy endpoints are considered
2. If there are enabled & healthy endpoints with different priorities, only the highest priority ones are considered
3. If there are highest-priority endpoints in multiple sites, one of them being the client's local site,
   only local site endpoints are returned, otherwise all sites' endpoints are returned
4. The endpoints are returned in random order (if there are more than 3 candidates, random 3 are returned)


Configuration
-------------

TODO


API Example
-----------

For full API reference see: https://goslb.docs.apiary.io/
