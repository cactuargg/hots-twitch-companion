package twitch

import (
 "context"
 "errors"
 "net/url"
 "testing"
 "time"
)

func TestDevicePollRetryBackoff(t *testing.T) {
 if next,retry:=DevicePollRetry(5*time.Second,errors.New("authorization_pending"));!retry||next!=5*time.Second{t.Fatalf("pending %s %v",next,retry)}
 if next,retry:=DevicePollRetry(5*time.Second,errors.New("slow_down"));!retry||next!=10*time.Second{t.Fatalf("slowdown %s %v",next,retry)}
 if _,retry:=DevicePollRetry(5*time.Second,errors.New("invalid_client"));retry{t.Fatal("terminal error retried")}
}

func TestDeviceStartUsesInjectedRequest(t *testing.T) {
 calls:=0
 req:=func(ctx context.Context,method,path string,form url.Values,body any,token,clientID string,dst any)error{
  calls++
  if method!="POST"||path!="https://example.test/oauth/device"||form.Get("client_id")!="app"||form.Get("scopes")!="channel:manage:predictions"||token!=""||clientID!=""{t.Fatalf("unexpected device request: %s %s %v",method,path,form)}
  d,ok:=dst.(*DeviceResponse)
  if !ok {t.Fatal("wrong response type")}
  *d=DeviceResponse{DeviceCode:"device",Interval:5}
  return nil
 }
 d,err:=StartDevice(context.Background(),"app","https://example.test/oauth","channel:manage:predictions",req)
 if err!=nil||d.DeviceCode!="device"||calls!=1{t.Fatalf("device flow request failed: %+v %v %d",d,err,calls)}
}
