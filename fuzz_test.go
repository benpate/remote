package remote

import (
	"encoding/json"
	"testing"
)

// FuzzTransactionUnmarshalJSON confirms that decoding arbitrary bytes into a
// Transaction never panics. Malformed input should return an error, not crash.
func FuzzTransactionUnmarshalJSON(f *testing.F) {

	f.Add([]byte(`{"method":"GET","url":"http://x.com"}`))
	f.Add([]byte(`{"header":{"Accept":"text/plain"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"query":"a=1&b=2"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		tx := New()
		_ = json.Unmarshal(data, tx)
	})
}

// FuzzAssembleBearCap confirms the BearCap URL parser never panics on arbitrary
// URL strings.
func FuzzAssembleBearCap(f *testing.F) {

	f.Add("bear:?t=token&u=http://target.com")
	f.Add("bear:?u=http://target.com")
	f.Add("bear:?t=token")
	f.Add("http://normal.com")
	f.Add("bear:?%zz")

	f.Fuzz(func(t *testing.T, url string) {
		tx := Get(url)
		_ = tx.assembleBearCap()
	})
}
