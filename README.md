### Usage
AB Apache tool for benchmarking. In this sample tests 50 concurrent requests

```
ab -c 50 -n 50 -X 127.0.0.1:8081 http://en.wikipedia.org/wiki/Perugia
```

Curl. Single request
```
curl -x http://localhost:8081 -k "http://en.wikipedia.org/wiki/Bologna"
```

In both expamples `-x` stands for proxy.

### TODOS
* copy headers

### Problems
* Handle HTTPS