package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

type ErrorFunc func(*http.Response, []byte) error
type AuthFunc func(*http.Request) error

var (
	debug = false
)

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL

	errorFunc ErrorFunc
	authFunc  AuthFunc
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	c := &Client{
		httpClient: httpClient,
	}
	c.baseURL, _ = url.Parse(baseURL)

	return c
}

func (c *Client) RegisterErrorFunc(ef ErrorFunc) {
	c.errorFunc = ef
}

func (c *Client) RegisterAuthFunc(af AuthFunc) {
	c.authFunc = af
}

func (c *Client) buildURL(s string, opt interface{}) (*url.URL, error) {
	u, err := c.baseURL.Parse(s)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return u, nil
	}

	qs, err := query.Values(opt)
	if err != nil {
		return u, err
	}

	u.RawQuery = qs.Encode()
	return u, nil
}

func buildBody(body interface{}) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	err := enc.Encode(body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Type and value of body must be nil for an empty body
func (c *Client) NewRequest(method, urlStr string, body interface{}, query interface{}) (*http.Request, error) {
	u, err := c.buildURL(urlStr, query)
	fmt.Printf("url: %+v\n", u)

	var b *bytes.Buffer
	if body != nil {
		b, err = buildBody(body)
		if err != nil {
			return nil, err
		}
	}

	// net/http checks type to parse the body; a nil bytes.Buffer causes a segfault
	var req *http.Request
	if b == nil {
		req, err = http.NewRequest(method, u.String(), nil)
	} else {
		req, err = http.NewRequest(method, u.String(), b)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	err = c.authFunc(req)
	if err != nil {
		return nil, err
	}

	if debug {
		requestDump, err := httputil.DumpRequest(req, true)
		if err == nil {
			fmt.Println(string(requestDump))
		}
	}

	return req, nil
}

func (c *Client) Do(req *http.Request, jsonResp interface{}) (*http.Response, error) {
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewAPIError(response.Request.RequestURI, "<pre response parse>", response.StatusCode, err)
	}

	if debug {
		responseDump, err := httputil.DumpResponse(response, true)
		if err == nil {
			fmt.Println(string(responseDump))
		}
	}

	/* HTTP lib error */
	if err != nil {
		if response != nil && response.Request != nil {
			return nil, NewAPIError(response.Request.RequestURI, "<pre response parse>", response.StatusCode, err)
		} else {
			return nil, NewAPIError("", "", 0, err)
		}
	}
	defer response.Body.Close()

	var byteResponse []byte
	byteResponse, err = ioutil.ReadAll(response.Body)
	if err != nil { /* body read error */
		return nil, NewAPIError(response.Request.URL.EscapedPath(), string(byteResponse), response.StatusCode, err)
	}

	if jsonResp != nil {
		err = json.Unmarshal(byteResponse, jsonResp)
	}
	if c.errorFunc != nil {
		err = c.errorFunc(response, byteResponse)
	}
	if err != nil {
		return nil, NewAPIError(response.Request.URL.EscapedPath(), string(byteResponse), response.StatusCode, err)
	}

	return response, nil
}
