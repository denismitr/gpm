package proxy

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
)

func TestURLArgumentParser(t *testing.T) {
	invalid := []struct {
		url string
		msg string
	}{
		{"url=" + uriEncode("http//google.com"), "passed url value does not match a valid url pattern"},
		{"url=google", "passed url value does not match a valid url pattern"},
		{"", "no [url] query param found"},
		{"url=" + uriEncode("/foo/bar"), "passed url value does not match a valid url pattern"},
		{"url=", "[url] query param is empty"},
	}

	valid := []struct {
		url    string
		result string
	}{
		{"url=" + uriEncode("http://google.com"), "http://google.com"},
		{"url=" + uriEncode("https://google.com?search=boo"), "https://google.com?search=boo"},
		{"api_key=secret&url=" + uriEncode("https://google.com?search=boo&cache=bust"), "https://google.com?search=boo&cache=bust"},
		{"api_key=secret&url=" + base64encode("https://google.com?search=boo&cache=bust"), "https://google.com?search=boo&cache=bust"},
	}

	for _, v := range invalid {
		t.Run("invalid URLs", func(t *testing.T) {
			r, _ := http.NewRequest("GET", "http://localhost:8081/get?"+v.url, nil)

			_, err := ParseURLParam(r)
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

			result, err := ParseURLParam(r)
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

func uriEncode(in string) string {
	return url.QueryEscape(in)
}
