package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRequestJSONHeadersAndDecoding(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		if r.Method!="POST" || r.Header.Get("Authorization")!="Bearer secret" ||
			r.Header.Get("Client-Id")!="app-client" || r.Header.Get("Content-Type")!="application/json" {
			t.Errorf("unexpected request: %s %v",r.Method,r.Header)
		}
		raw,_:=io.ReadAll(r.Body)
		if string(raw)!=`{"value":"hello"}` {t.Errorf("unexpected JSON: %s",raw)}
		w.Header().Set("Content-Type","application/json")
		_,_ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	var response struct{OK bool `json:"ok"`}
	err:=RequestJSON(context.Background(),server.Client(),"POST",server.URL,nil,map[string]string{"value":"hello"},
		"secret","app-client",&response,func(*http.Response,string)error{return errors.New("http error")})
	if err!=nil||!response.OK {t.Fatalf("wrong result %+v %v",response,err)}
}

func TestRequestJSONOAuthValidateAndHTTPError(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		if r.Header.Get("Authorization")!="OAuth token" {t.Errorf("expected OAuth validation header, got %q",r.Header.Get("Authorization"))}
		w.WriteHeader(http.StatusTooManyRequests)
		_,_ = w.Write([]byte(`{"message":"too many requests"}`))
	}))
	defer server.Close()
	seen:=false
	err:=RequestJSON(context.Background(),server.Client(),"GET",server.URL+"/validate",nil,nil,"token","",nil,
		func(r *http.Response,msg string)error{
			seen=true
			if r.StatusCode!=429||msg!="too many requests" {t.Errorf("bad typed error input: %d %q",r.StatusCode,msg)}
			return errors.New("rate limited")
		})
	if err==nil||!seen {t.Fatalf("expected typed HTTP error callback, got %v",err)}
}

func TestRequestJSONFormEncoding(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"),"application/x-www-form-urlencoded") {
			t.Errorf("incorrect content-type")
		}
		b,_:=io.ReadAll(r.Body)
		values,err:=url.ParseQuery(string(b))
		if err!=nil||values.Get("client_id")!="abc"{t.Errorf("form lost: %s %v",b,err)}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err:=RequestJSON(context.Background(),server.Client(),"POST",server.URL,url.Values{"client_id":{"abc"}},
		nil,"","",nil,func(*http.Response,string)error{return errors.New("failed")});err!=nil{t.Fatal(err)}
}
