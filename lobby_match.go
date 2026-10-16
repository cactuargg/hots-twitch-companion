package main

// LobbySnapshot is durable match provenance captured when this Companion instance
// observes a HotS loading screen. The binary battlelobby format is NOT parsed here;
// its SHA-256 is an identifier of this particular lobby file, not an identifier
// present in the later StormReplay. Do not infer an exact match from the hash.
import (
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "io"
 "os"
 "path/filepath"
 "time"
)

const maxLobbyBytes = 8 << 20

type LobbySnapshot struct {
 SHA256 string `json:"sha256"`
 File string `json:"file"`
 ModifiedAt int64 `json:"modifiedAt"`
 Size int64 `json:"size"`
}

func captureLobbySnapshot(path string) (*LobbySnapshot,error) {
 if path=="" {return nil,errors.New("no battlelobby path")}
 f,err:=os.Open(path)
 if err!=nil{return nil,err}
 defer f.Close()
 st,err:=f.Stat()
 if err!=nil{return nil,err}
 if !st.Mode().IsRegular()||st.Size()<1||st.Size()>maxLobbyBytes {
  return nil,errors.New("battlelobby size or type is not supported")
 }
 h:=sha256.New()
 n,err:=io.Copy(h,io.LimitReader(f,maxLobbyBytes+1))
 if err!=nil{return nil,err}
 if n!=st.Size(){return nil,errors.New("battlelobby modified while capturing")}
 current,err:=os.Stat(path)
 if err!=nil{return nil,err}
 if current.Size()!=st.Size()||!current.ModTime().Equal(st.ModTime()){
  return nil,errors.New("battlelobby changed while capturing")
 }
 return &LobbySnapshot{
  SHA256:hex.EncodeToString(h.Sum(nil)),
  File:filepath.Base(filepath.Dir(path)),
  ModifiedAt:st.ModTime().UnixMilli(), Size:st.Size(),
 },nil
}

// Temporal relation is necessary but not sufficient to prove game identity.
// Real cross-file match verification requires parsing lobby player ToonHandles
// and comparing them with the replay roster.
func (l *LobbySnapshot) replayCanFollow(replayModified time.Time) bool {
 if l==nil||l.ModifiedAt<=0 {return true}
 return !replayModified.Before(time.UnixMilli(l.ModifiedAt).Add(-2*time.Second))
}
