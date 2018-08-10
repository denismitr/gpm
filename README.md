### Usage
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