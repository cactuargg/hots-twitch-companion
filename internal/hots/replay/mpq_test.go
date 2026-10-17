package replay

import (
	"bytes"
	"testing"
)

func TestMPQRejectsOversizedAndMalformedArchives(t *testing.T) {
	for _, tc:=range [][]byte{
		nil, {}, []byte("broken"), bytes.Repeat([]byte{255},32),
		bytes.Repeat([]byte{0},MaxReplay+1),
	} {
		if value,err:=ReadMPQMember(tc,"replay.details");err==nil||len(value)>0 {
			t.Fatalf("malformed archive unexpectedly decoded: len=%d, result=%d, err=%v",len(tc),len(value),err)
		}
	}
}

func TestMPQCompressedMemberMissingHeader(t *testing.T) {
	if _,err:=UnpackCompressed(nil);err==nil{t.Fatal("empty compressed member accepted")}
	if _,err:=UnpackCompressed([]byte{0xff,0});err==nil{t.Fatal("unknown compression accepted")}
}
