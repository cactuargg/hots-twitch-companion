package storage

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadJSON(t *testing.T) {
	file := filepath.Join(t.TempDir(),"state.json")
	before := struct {
		Version int `json:"version"`
		Flags []string `json:"flags"`
	}{Version:3, Flags:[]string{"safe","enabled"}}
	if err:=SaveJSON(file,before);err!=nil {t.Fatal(err)}
	var restored struct {
		Version int `json:"version"`
		Flags []string `json:"flags"`
	}
	if err:=LoadJSON(file,&restored);err!=nil{t.Fatal(err)}
	if restored.Version!=before.Version || !reflect.DeepEqual(restored.Flags,before.Flags) {
		t.Fatalf("roundtrip failed: %+v",restored)
	}
	if _,err:=os.Stat(file+".new");!errors.Is(err,os.ErrNotExist){t.Fatalf("temp file remained: %v",err)}
}

func TestSaveJSONMarshalErrorPreservesPriorState(t *testing.T) {
	file:=filepath.Join(t.TempDir(),"state.json")
	if err:=SaveJSON(file,map[string]string{"status":"committed"});err!=nil {t.Fatal(err)}
	if err:=SaveJSON(file,map[string]any{"invalid":make(chan int)});err==nil {
		t.Fatal("marshal of unsupported type was accepted")
	}
	var saved map[string]string
	if err:=LoadJSON(file,&saved);err!=nil {t.Fatal(err)}
	if saved["status"]!="committed" {t.Fatalf("failed save damaged committed state: %+v",saved)}
}

func TestLoadJSONMissingFileAndInvalidData(t *testing.T) {
	file:=filepath.Join(t.TempDir(),"missing.json")
	var saved map[string]string
	if err:=LoadJSON(file,&saved);err!=nil {t.Fatal(err)}
	if saved!=nil {t.Fatalf("unexpected default state: %+v",saved)}
	if err:=os.WriteFile(file,[]byte("{broken"),0600);err!=nil {t.Fatal(err)}
	if err:=LoadJSON(file,&saved);err==nil {t.Fatal("corrupt state was accepted")}
}
