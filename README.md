## Go proxy mutiplexer
The proxy works as an api receiving requests in format
`/get?api_key=secret&url=httpsbin.org/html` and returns the response body and headers

### Dependencies
* `go get -u github.com/go-chi/chi`
* `go get -u github.com/joho/godotenv`

### Environment variables required for the proxy server to work
* `GPM_PORT` - port on wich the microservice works (defaults to `:8081`)
* `GPM_SERVER_API_KEY` - security api key
* `GPM_PROXY_LIST` - file that contains the list of proxy servers, can be a relative 
or an absolute path. Defaults to "proxy.list"
* `GPM_CONCURRENT_TRIES` - how many concurrent request through proxy service is going to be made concurrently (defaults to 3)
* `GPM_MAX_TIMEOUT` - maximum timeout after which an error response ig going to be send (defaults to 10 seconds)

### Usage (this functionality is temporarily disabled)
To make api_key mandatory just set `GPM_SERVER_API_KEY` to some value e.g. `export GPM_SERVER_API_KEY=secret`

To use proxy service fill a specified in  `GPM_PROXY_LIST` file with proxies you want to use.

Example:
```
127.0.0.1:8089
127.0.0.1:8088
```


Curl. Single request
```
curl "http://localhost:8081/get?url=https://httpbin.org/json"
curl "http://localhost:8081/get?url=https://httpbin.org/html"
```

#### Benchmarking
AB Apache tool for benchmarking. In this sample tests 50 concurrent requests

```
ab -c 16 -n 16 "127.0.0.1:8081/get?api_key=secret&url=https://httpbin.org/html"
```