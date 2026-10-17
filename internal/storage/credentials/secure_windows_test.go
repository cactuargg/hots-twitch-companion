//go:build windows

package credentials

import (
  "bytes"
  "crypto/rand"
  "testing"
)

func TestDPAPIRoundtripAndNonPlaintext(t *testing.T) {
  plain:=make([]byte,128)
  if _,err:=rand.Read(plain);err!=nil{t.Fatal(err)}
  encrypted,err:=Protect(plain)
  if err!=nil{t.Fatal(err)}
  if bytes.Equal(encrypted,plain)||len(encrypted)==0 {t.Fatal("DPAPI failed to protect plaintext")}
  restored,err:=Unprotect(encrypted)
  if err!=nil{t.Fatal(err)}
  if !bytes.Equal(restored,plain) {t.Fatal("DPAPI decrypt changed credential bytes")}
  if _,err:=Unprotect([]byte("corrupt DPAPI bytes"));err==nil {t.Fatal("invalid DPAPI blob was accepted")}
}
