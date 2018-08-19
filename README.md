## Refactor in progress
The proxy will now work as an api receiving requests in format
`/get?url=httpsbin.org/html` and return the response body and headers

### Environment variables required for the proxy server to work
* `GPM_SERVER_API_KEY` - security api key
* `GPM_PROXY_URL` - third party proxy service url
* `GPM_PROXY_AUTH` -  third party proxy service auth token
* `GPM_CONCURRENT_TRIES` - how many concurrent request through proxy service is going to be made concurrently
* `GPM_MAX_TIMEOUT` - maximum timeout after which an error response ig going to be send

### Usage (this functionality is temporarily disabled)
Run `main.go` in a console, then in another console use curl like in below examples.


Curl. Single request
```
curl -x http://localhost:8081 -k "http://en.wikipedia.org/wiki/Bologna"
```

#### Benchmarking
AB Apache tool for benchmarking. In this sample tests 50 concurrent requests

```
ab -c 50 -n 50 -X 127.0.0.1:8081 http://en.wikipedia.org/wiki/Perugia
```

In both expamples `-x` stands for proxy.

### Todos
Right now it uses some free proxy found on google. Next step is to actually 
use some good proxy service.

### Problems
* Handle HTTPS
