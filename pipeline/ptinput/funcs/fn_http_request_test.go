package funcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GuanceCloud/cliutils/pipeline/ptinput"
	"github.com/GuanceCloud/cliutils/point"
	"github.com/stretchr/testify/assert"
)

func TestBuildBody(t *testing.T) {
	cases := []struct {
		val    any
		result string
	}{
		{
			val:    float64(123.1),
			result: "123.1",
		},
		{
			val:    int64(123),
			result: "123",
		},
		{
			val:    true,
			result: "true",
		},
		{
			val:    false,
			result: "false",
		},
		{
			val:    "abc",
			result: "abc",
		},
		{
			val:    []any{1, 2, 3},
			result: "[1,2,3]",
		},
		{
			val:    map[string]any{"a": 1, "b": 2},
			result: `{"a":1,"b":2}`,
		},
		{
			val:    nil,
			result: "",
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("index_%d", i), func(t *testing.T) {
			var buf []byte
			if b := buildBody(c.val); b != nil {
				var err error
				buf, err = io.ReadAll(b)
				if err != nil && !errors.Is(err, io.EOF) {
					t.Error(err)
				}
			}
			assert.Equal(t, c.result, string(buf))
		})
	}
}

func TestHTTPRequest(t *testing.T) {
	server := HTTPServer()
	defer server.Close()

	url := `"` + server.URL + "/testResp" + `"`
	fmt.Println(url)

	cases := []struct {
		name, pl, in string
		expected     interface{}
		fail         bool
		outkey       string
	}{
		{
			name: "test_post",
			pl: fmt.Sprintf(`
			resp = http_request("POST", %s, {"extraHeader": "1", 
			"extraHeader": "1"}, {"a": "1"})
			add_key(abc, resp["body"])	
			`, url),
			in:       `[]`,
			outkey:   "abc",
			expected: `{"a":"1"}`,
		},
		{
			name: "test_put",
			pl: fmt.Sprintf(`
			resp = http_request("put", %s, {"extraHeader": "1", 
			"extraHeader": "1"}, {"a": "1"})
			add_key(abc, resp["body"])	
			`, url),
			in:       `[]`,
			outkey:   "abc",
			expected: `{"a":"1"}`,
		},
	}

	for idx, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := NewTestingRunner(tc.pl)
			if err != nil {
				if tc.fail {
					t.Logf("[%d]expect error: %s", idx, err)
				} else {
					t.Errorf("[%d] failed: %s", idx, err)
				}
				return
			}
			pt := ptinput.NewPlPoint(
				point.Logging, "test", nil, map[string]any{"message": tc.in}, time.Now())
			errR := runScript(runner, pt)

			if errR != nil {
				t.Fatal(errR.Error())
			}

			v, _, _ := pt.Get(tc.outkey)
			// tu.Equals(t, nil, err)
			assert.Equal(t, tc.expected, v)

			t.Logf("[%d] PASS", idx)
		})
	}
}

func HTTPServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			headers := r.Header

			var respData []byte
			var err error
			if headers.Get("extraHeader1") != "" && headers.Get("extraHeader2") != "" {
				responseData := map[string]string{"a": "hello world"}
				respData, err = json.Marshal(responseData)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			} else {
				switch r.Method {
				case http.MethodGet:
					responseData := map[string]string{"a": "hello"}
					respData, err = json.Marshal(responseData)
					if err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
				default:
					d, _ := io.ReadAll(r.Body)
					respData = d
				}
			}

			w.Write(respData)
			w.WriteHeader(http.StatusOK)
		},
	))
	return server
}
