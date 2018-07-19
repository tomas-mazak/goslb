GoSLB - DNS load balancer
=========================

GoSLB is multi-site DNS load-balancer with a RESTful API and etcd backend. The basic configuration
unit is a _service_ (DNS domain) with a list of _endpoints_ (IP addresses) and a TCP or a HTTP 
health _monitor_. Each endpoint can be assigned to a _site_ (datacenter, cloud region, ...) and,
based on provided site's IP ranges, local-site endpoints are preferred in the DNS responses to 
clients.
