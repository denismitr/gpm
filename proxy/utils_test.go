package proxy

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestURLArgumentParser(t *testing.T) {
	invalid := []struct {
		url string
		msg string
	}{
		{"url=" + base64encode("http//google.com"), "Passed url value does not match a valid url pattern"},
		{"url=google", "Passed url value does not match a valid url pattern"},
		{"", "No [url] query param found"},
		{"url=" + base64encode("/foo/bar"), "Passed url value does not match a valid url pattern"},
		{"url=", "[url] query param is empty"},
	}

	valid := []struct {
		url    string
		result string
	}{
		{"url=" + base64encode("http://google.com"), "http://google.com"},
		{"url=" + base64encode("https://google.com?search=boo"), "https://google.com?search=boo"},
		{"api_key=secret&url=" + base64encode("https://google.com?search=boo&cache=bust"), "https://google.com?search=boo&cache=bust"},
	}

	for _, v := range invalid {
		t.Run("invalid URLs", func(t *testing.T) {
			r, _ := http.NewRequest("GET", "http://localhost:8081/get?"+v.url, nil)

			_, err := ParseURLArgument(r)
			if err == nil {
				t.Fatalf("Expeced an error on url %s, but did not get one", v.url)
			}

			if err.Error() != v.msg {
				t.Fatalf("Expected error %s but got %s for url %s", v.msg, err.Error(), v.url)
			}
		})
	}

	for _, v := range valid {
		t.Run("invalid URLs", func(t *testing.T) {
			r, _ := http.NewRequest("GET", "http://localhost:8081/get?"+v.url, nil)

			result, err := ParseURLArgument(r)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}

			if result != v.result {
				t.Fatalf("Expected result %s but got %s", v.result, result)
			}
		})
	}
}

func base64encode(in string) string {
	return base64.StdEncoding.EncodeToString([]byte(in))
}
