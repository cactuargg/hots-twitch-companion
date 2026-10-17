//go:build windows

package credentials

import (
 "bytes"
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "testing"
)

func TestProtectedTokenRoundTrip(t *testing.T){
 path:=filepath.Join(t.TempDir(),"tokens.dpapi")
 token:=struct{Access string `json:"access_token"`;Refresh string `json:"refresh_token"`}{Access:"test-access-secret",Refresh:"test-refresh-secret"}
 if err:=SaveJSON(path,token);err!=nil{t.Fatal(err)}
 encrypted,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 original,_:=json.Marshal(token)
 if bytes.Equal(encrypted,original)||bytes.Contains(encrypted,[]byte("test-access-secret")){t.Fatal("token was written in plaintext")}
 var back struct{Access string `json:"access_token"`;Refresh string `json:"refresh_token"`}
 if err:=LoadJSON(path,&back);err!=nil{t.Fatal(err)}
 if back.Access!=token.Access||back.Refresh!=token.Refresh{t.Fatalf("token modified: %+v",back)}
 if _,err:=os.Stat(path+".new");!errors.Is(err,os.ErrNotExist){t.Fatal("temporary token file remained")}
}

func TestCorruptEncryptedTokenIsRejectedWithoutOverwriting(t *testing.T){
 path:=filepath.Join(t.TempDir(),"tokens.dpapi")
 corrupt:=[]byte("bad dpapi")
 if err:=os.WriteFile(path,corrupt,0600);err!=nil{t.Fatal(err)}
 var v map[string]string
 if err:=LoadJSON(path,&v);err==nil{t.Fatal("corrupt encrypted token was accepted")}
 after,err:=os.ReadFile(path);if err!=nil||!bytes.Equal(after,corrupt){t.Fatal("corrupt token was overwritten")}
 if err:=SaveJSON(path,map[string]any{"invalid":make(chan int)});err==nil{t.Fatal("invalid JSON was accepted")}
 after,_=os.ReadFile(path)
 if !bytes.Equal(after,corrupt){t.Fatal("failed save overwrote token")}
}
