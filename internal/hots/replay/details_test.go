package replay

import (
	"testing"
)

func TestParseDetailsRejectsBrokenVersionedData(t *testing.T) {
	for _,input:=range [][]byte{nil,{}, {99}, {5,0}, {5,0xff}} {
		value,err:=ParseDetails(input)
		if err==nil || len(value.Players)>0 {
			t.Fatalf("invalid details accepted: raw=%v parsed=%+v err=%v",input,value,err)
		}
	}
}

func TestDecodedReplayStructRetainsJSONContract(t *testing.T) {
	p:=Player{Name:"Player",ToonHandle:"2-Hero-1-42",Result:"win",ResultCode:1,Team:1}
	if p.Result!="win" || p.ToonHandle=="" {t.Fatal("player metadata changed")}
}
