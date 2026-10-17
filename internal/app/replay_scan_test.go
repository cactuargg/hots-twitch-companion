package app

import (
 "encoding/json"
 "strings"
 "testing"
)

func TestReplayScanJSONContract(t *testing.T) {
 data, err := json.Marshal(ReplayScan{Code:"matched",File:"match.StormReplay",CheckedAt:42})
 if err != nil { t.Fatal(err) }
 for _, key := range []string{`"code":"matched"`,`"file":"match.StormReplay"`,`"checkedAt":42`} {
  if !strings.Contains(string(data),key) {t.Fatalf("missing %s in %s",key,data)}
 }
 if strings.Contains(string(data),`"detail"`) {t.Fatalf("empty detail must be omitted: %s",data)}
 var decoded ReplayScan
 if err:=json.Unmarshal(data,&decoded);err!=nil {t.Fatal(err)}
 if decoded.Code!="matched" || decoded.CheckedAt!=42 {t.Fatalf("unexpected decoded scan: %+v",decoded)}
}
