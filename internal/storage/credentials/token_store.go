package credentials

import (
 "encoding/json"
 "fmt"
 "os"
)

// LoadJSON decrypts a previously persisted DPAPI payload and decodes its JSON.
// Missing files remain distinguishable using errors.Is(err, os.ErrNotExist).
// A corrupt protected blob never triggers creation of a blank credential file.
func LoadJSON(path string,dst any) error{
 data,err:=os.ReadFile(path)
 if err!=nil{return err}
 clear,err:=Unprotect(data)
 if err!=nil{return fmt.Errorf("Windows DPAPI: %w",err)}
 return json.Unmarshal(clear,dst)
}

// SaveJSON preserves the application's .new+rename credential replacement
// semantics. The caller changes its in-memory token only after success.
func SaveJSON(path string,value any) error{
 plain,err:=json.Marshal(value)
 if err!=nil{return err}
 sealed,err:=Protect(plain)
 if err!=nil{return err}
 if err:=os.WriteFile(path+".new",sealed,0600);err!=nil{return err}
 return os.Rename(path+".new",path)
}
