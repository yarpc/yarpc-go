This is an example that illustrates an HTTP to TChannel transport translation
proxy for a single service in a Docker composition.
The "caller" services sends HTTP requests, the "proxy" service receives HTTP
requests and forwards them over TChannel to the "service".

To run this example, make the Linux binaries for each of the services
(`*/main.go`) and then bring up the services in Docker.

```
make
docker-compose up
```

The output should contain this repeating sequence of logs:

```
caller_1   | 2016/11/22 19:50:07 sending request
proxy_1    | 2016/11/22 19:50:07 forwarding request from caller to service to Service::procedure
service_1  | 2016/11/22 19:50:07 handling request
caller_1   | 2016/11/22 19:50:07 received response
```

The demo also runs locally as three concurrent processes.
The service listens for TChannel requests on some port.

```
cd server
go run main.go :4040
```

The proxy listens for HTTP requests and forwards to the TChannel service.

```
cd proxy
go run main.go :8080 127.0.0.1:4040
```

The caller sends period requests over HTTP to the proxy.

```
cd caller
go run main.go http://127.0.0.1:8080
```
