//go:build !windows

package credentials

import "testing"

func TestNoDPAPIOutsideWindows(t *testing.T){
 if _,err:=Protect([]byte("secret"));err==nil{t.Fatal("credentials unprotected outside Windows")}
 if _,err:=Unprotect([]byte("encrypted"));err==nil{t.Fatal("Windows DPAPI format treated as plain text")}
}
