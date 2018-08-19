## Refactor in progress
The proxy will now work as an api receiving requests in format
`/get?url=httpsbin.org/html` and return the response body and headers

### Environment variables required for the proxy server to work
* `GPM_PORT` - port on wich the microservice works (defaults to :8081)
* `GPM_SERVER_API_KEY` - security api key
* `GPM_PROXY_URL` - third party proxy service url
* `GPM_PROXY_AUTH` -  third party proxy service auth token
* `GPM_CONCURRENT_TRIES` - how many concurrent request through proxy service is going to be made concurrently (defaults to 3)
* `GPM_MAX_TIMEOUT` - maximum timeout after which an error response ig going to be send (defaults to 10 seconds)

### Usage (this functionality is temporarily disabled)
Curl. Single request
```
curl "http://localhost:8081/get?url=https://httpbin.org/json"
curl "http://localhost:8081/get?url=https://httpbin.org/html"
```

#### Benchmarking
AB Apache tool for benchmarking. In this sample tests 50 concurrent requests

```
ab -c 50 -n 50 -X 127.0.0.1:8081 http://en.wikipedia.org/wiki/Perugia
```