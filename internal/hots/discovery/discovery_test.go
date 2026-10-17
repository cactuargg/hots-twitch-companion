package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplayInventoryOrderingAndRoot(t *testing.T) {
	root:=t.TempDir()
	now:=time.Now()
	paths:=[]string{
		filepath.Join(root,"old.StormReplay"),
		filepath.Join(root,"new.stormreplay"),
		filepath.Join(root,"unrelated.txt"),
	}
	for i,p:=range paths {
		if err:=os.WriteFile(p,[]byte("replay"),0600);err!=nil{t.Fatal(err)}
		at:=now.Add(time.Duration(i)*time.Second)
		if err:=os.Chtimes(p,at,at);err!=nil{t.Fatal(err)}
	}
	files:=ReplaysAfter(root,now.Add(-time.Second),12)
	if len(files)!=2 || filepath.Base(files[0].Path)!="new.stormreplay" ||
		filepath.Base(files[1].Path)!="old.StormReplay" {
		t.Fatalf("replay filtering/order changed: %+v",files)
	}
	roots:=ReplayRoots(root)
	if len(roots)!=1||roots[0]!=root{t.Fatalf("custom root changed: %v",roots)}
	if len(ReplayRoots(""))!=2{t.Fatal("default replay roots changed")}
}
