package dashboard

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestHTTPRouterCapabilityAndOrigin(t *testing.T) {
 called,active:=0,0
 h:=NewHTTPHandler("127.0.0.1:1234","secret",func(w http.ResponseWriter,r *http.Request,route string){called++;w.WriteHeader(204)},func(){active++})
 tests:=[]struct{name,path,host,method,header,origin,site string;status int}{
  {"wrong host","/secret/api/status","evil.example","GET","1","","",403},
  {"no capability","/api/status","127.0.0.1:1234","GET","1","","",404},
  {"missing header","/secret/api/status","127.0.0.1:1234","GET","","","",403},
  {"bad origin","/secret/api/status","127.0.0.1:1234","POST","1","https://evil.example","",403},
  {"cross site","/secret/api/status","127.0.0.1:1234","POST","1","","cross-site",403},
  {"valid get","/secret/api/status","127.0.0.1:1234","GET","1","","",204},
  {"valid post","/secret/api/status","127.0.0.1:1234","POST","1","http://127.0.0.1:1234","same-origin",204},
  {"web ui","/secret/","127.0.0.1:1234","GET","","","",200},
 }
 for _,tt:=range tests {
  t.Run(tt.name,func(t *testing.T){
   req:=httptest.NewRequest(tt.method,tt.path,nil);req.Host=tt.host
   if tt.header!="" {req.Header.Set("X-Companion",tt.header)}
   if tt.origin!="" {req.Header.Set("Origin",tt.origin)}
   if tt.site!="" {req.Header.Set("Sec-Fetch-Site",tt.site)}
   w:=httptest.NewRecorder();h.ServeHTTP(w,req)
   if w.Code!=tt.status {t.Fatalf("status=%d want=%d",w.Code,tt.status)}
   if tt.name=="valid get" && !strings.Contains(w.Header().Get("Content-Security-Policy"),"default-src") {t.Fatal("missing CSP")}
  })
 }
 if called!=2 || active!=6 {t.Fatalf("callbacks: api=%d activity=%d",called,active)}
}
