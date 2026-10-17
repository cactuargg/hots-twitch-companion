package twitch

import (
	"encoding/json"
	"testing"
)

func TestOAuthAndPredictionJSONContract(t *testing.T) {
	var token Token
	if err:=json.Unmarshal([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":3600,"savedAt":1234}`),&token);err!=nil {t.Fatal(err)}
	if token.AccessToken!="access"||token.RefreshToken!="refresh"||token.SavedAt!=1234 {t.Fatalf("token format changed: %+v",token)}
	var remote APIResponse
	sample:=`{"data":[{"id":"123","status":"ACTIVE","winning_outcome_id":"","outcomes":[{"id":"w","title":"Win"},{"id":"l","title":"Loss"}]}]}`
	if err:=json.Unmarshal([]byte(sample),&remote);err!=nil{t.Fatal(err)}
	if len(remote.Data)!=1||len(remote.Data[0].Outcomes)!=2||remote.Data[0].Outcomes[0].ID!="w" {
		t.Fatalf("Twitch API decoded differently: %+v",remote)
	}
	var me Identity
	if err:=json.Unmarshal([]byte(`{"client_id":"app","user_id":"channel","login":"streamer","scopes":["channel:manage:predictions"]}`),&me);err!=nil{t.Fatal(err)}
	if me.UserID!="channel"||me.ClientID!="app"||len(me.Scopes)!=1 {t.Fatalf("identity fields changed: %+v",me)}
}
