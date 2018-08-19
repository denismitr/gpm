package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi"
)

const testHTML = `<!DOCTYPE html>
<html>
  <head>
  </head>
  <body>
      <h1>Herman Melville - Moby-Dick</h1>

      <div>
        <p>
          Availing himself of the mild, summer-cool weather that now reigned in these latitudes, and in preparation for the peculiarly active pursuits shortly to be anticipated, Perth, the begrimed, blistered old blacksmith, had not removed his portable forge to the hold again, after concluding his contributory work for Ahab's leg, but still retained it on deck, fast lashed to ringbolts by the foremast; being now almost incessantly invoked by the headsmen, and harpooneers, and bowsmen to do some little job for them; altering, or repairing, or new shaping their various weapons and boat furniture. Often he would be surrounded by an eager circle, all waiting to be served; holding boat-spades, pike-heads, harpoons, and lances, and jealously watching his every sooty movement, as he toiled. Nevertheless, this old man's was a patient hammer wielded by a patient arm. No murmur, no impatience, no petulance did come from him. Silent, slow, and solemn; bowing over still further his chronically broken back, he toiled away, as if toil were life itself, and the heavy beating of his hammer the heavy beating of his heart. And so it was.—Most miserable! A peculiar walk in this old man, a certain slight but painful appearing yawing in his gait, had at an early period of the voyage excited the curiosity of the mariners. And to the importunity of their persisted questionings he had finally given in; and so it came to pass that every one now knew the shameful story of his wretched fate. Belated, and not innocently, one bitter winter's midnight, on the road running between two country towns, the blacksmith half-stupidly felt the deadly numbness stealing over him, and sought refuge in a leaning, dilapidated barn. The issue was, the loss of the extremities of both feet. Out of this revelation, part by part, at last came out the four acts of the gladness, and the one long, and as yet uncatastrophied fifth act of the grief of his life's drama. He was an old man, who, at the age of nearly sixty, had postponedly encountered that thing in sorrow's technicals called ruin. He had been an artisan of famed excellence, and with plenty to do; owned a house and garden; embraced a youthful, daughter-like, loving wife, and three blithe, ruddy children; every Sunday went to a cheerful-looking church, planted in a grove. But one night, under cover of darkness, and further concealed in a most cunning disguisement, a desperate burglar slid into his happy home, and robbed them all of everything. And darker yet to tell, the blacksmith himself did ignorantly conduct this burglar into his family's heart. It was the Bottle Conjuror! Upon the opening of that fatal cork, forth flew the fiend, and shrivelled up his home. Now, for prudent, most wise, and economic reasons, the blacksmith's shop was in the basement of his dwelling, but with a separate entrance to it; so that always had the young and loving healthy wife listened with no unhappy nervousness, but with vigorous pleasure, to the stout ringing of her young-armed old husband's hammer; whose reverberations, muffled by passing through the floors and walls, came up to her, not unsweetly, in her nursery; and so, to stout Labor's iron lullaby, the blacksmith's infants were rocked to slumber. Oh, woe on woe! Oh, Death, why canst thou not sometimes be timely? Hadst thou taken this old blacksmith to thyself ere his full ruin came upon him, then had the young widow had a delicious grief, and her orphans a truly venerable, legendary sire to dream of in their after years; and all of them a care-killing competency.
        </p>
      </div>
  </body>
</html>`

func TestProxyGetRequest(t *testing.T) {
	t.Parallel()

	os.Setenv("GPM_SERVER_API_KEY", "secret")

	t.Run("html 200 response", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.ProxyGetRequest)

		testBody := []byte(testHTML)

		r.Get("/get", testHandler(t, r, server, testBody))

		ts := httptest.NewServer(r)
		defer ts.Close()

		uri := "/get?url=" + uriEncode("https://httpbin.org/html")

		_, body := testRequest(t, ts, "GET", uri, nil)
		if reflect.DeepEqual(body, testBody) {
			t.Fatalf("Expected test body, got %v", body)
		}
	})

	t.Run("json response", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.ProxyGetRequest)

		data, err := ioutil.ReadFile("../testdata/response_1534688291588.json")
		if err != nil {
			t.Fatalf("Could not read test data from response_1534688291588.json")
		}

		testBody := []byte(data)

		r.Get("/get", testJSONHandler(t, r, server, testBody))

		ts := httptest.NewServer(r)
		defer ts.Close()

		uri := "/get?url=" + uriEncode("https://httpbin.org/json")

		_, body := testRequest(t, ts, "GET", uri, nil)
		if reflect.DeepEqual(body, testBody) {
			t.Fatalf("Expected test body, got %v", body)
		}
	})

	t.Run("500 response", func(t *testing.T) {
		// Given
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.ProxyGetRequest)

		r.Get("/get", testErrorHandler(t, r, server, errors.New("error status 500 received from https://httpbin.org/status/500")))

		ts := httptest.NewServer(r)
		defer ts.Close()

		_, body := testRequest(t, ts, "GET", "/get?url=https://httpbin.org/status/500", nil)

		if !strings.Contains(string(body), "error status 500 received from https://httpbin.org/status/500") {
			t.Fatalf("Expected valid error string, got %v", body)
		}
	})

	t.Run("invalid url argument", func(t *testing.T) {
		// Given
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.ProxyGetRequest)
		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should have never reached this handler")
		})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get?url=wrong", nil)

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected bad request, got %s", resp.Status)
		}
	})

	os.Setenv("GPM_SERVER_API_KEY", "")
}

func TestCheckAPIKey(t *testing.T) {
	os.Setenv("GPM_SERVER_API_KEY", "secret")

	t.Run("correct api key", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.CheckAPIKey)

		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get?api_key=secret", nil)

		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatal("Received unauthorized response on correct api key")
		}
	})

	t.Run("incorrect api key", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.CheckAPIKey)

		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("Should not have gotten here")
		})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get?api_key=incorrect", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("Did not received unauthorized response on incorrect api key")
		}
	})

	t.Run("empty api key passed", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.CheckAPIKey)

		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("Should not have gotten here")
		})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get?api_key=", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("Did not received unauthorized response on incorrect api key")
		}
	})

	t.Run("no api key passed", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.CheckAPIKey)

		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("Should not have gotten here")
		})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("Did not received unauthorized response on incorrect api key")
		}
	})

	os.Setenv("GPM_SERVER_API_KEY", "")

	t.Run("no api key required", func(t *testing.T) {
		r := chi.NewRouter()

		logger := log.New(os.Stdout, "", log.LstdFlags)
		server := NewServer(logger)

		r.Use(server.CheckAPIKey)

		r.Get("/get", func(w http.ResponseWriter, r *http.Request) {})

		ts := httptest.NewServer(r)
		defer ts.Close()

		resp, _ := testRequest(t, ts, "GET", "/get?api_key=not-required", nil)

		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatal("Received unauthorized response when no api key required")
		}
	})
}

func testErrorHandler(t *testing.T, r *chi.Mux, s *Server, expectedError error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, ok := r.Context().Value(responseKey).(*FirstResponse)
		if !ok {
			t.Fatal("Could not find responseKey in request")
		}

		if response.IsValid() != false {
			t.Fatal("Expected response to be invalid but it is the opposite")
		}

		if response.GetError().Error() != expectedError.Error() {
			t.Fatalf("Expected error to be %v but got %v", expectedError, response.GetError())
		}

		s.proxyResponse(w, response)
	}
}

func testHandler(t *testing.T, r *chi.Mux, s *Server, expectedBody []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, ok := r.Context().Value(responseKey).(*FirstResponse)
		if !ok {
			t.Fatal("Could not find responseKey in request")
		}

		if response.IsValid() != true {
			t.Fatalf("Expected response to be valid but it is not")
		}

		defer response.CloseBody()

		body, _ := ioutil.ReadAll(response.GetBody())

		if len(body) != len(expectedBody) {
			t.Errorf("Expected response body to have length of %d instead got %d", len(expectedBody), len(body))
		}

		if !reflect.DeepEqual(body, expectedBody) {
			t.Errorf("Expected to see test body %s but got %s", string(expectedBody), string(body))
		}

		s.proxyResponse(w, response)
	}
}

func testJSONHandler(t *testing.T, r *chi.Mux, s *Server, expectedJSON []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, ok := r.Context().Value(responseKey).(*FirstResponse)
		if !ok {
			t.Fatal("Could not find responseKey in request")
		}

		if response.IsValid() != true {
			t.Fatalf("Expected response to be valid but it is not")
		}

		defer response.CloseBody()

		body, _ := ioutil.ReadAll(response.GetBody())

		assertExactJSON(t, expectedJSON, body)

		s.proxyResponse(w, response)
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

func assertExactJSON(t *testing.T, json1, json2 []byte) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal(json1, &o1)
	if err != nil {
		t.Fatalf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal(json2, &o2)
	if err != nil {
		t.Fatalf("Error mashalling string 2 :: %s", err.Error())
	}

	if !reflect.DeepEqual(o1, o2) {
		t.Fatalf("Failed asserting that two json structures are equal %v != %v", json1, json2)
	}
}
