package remote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeader(t *testing.T) {

	tx := Get("http://example.com")

	tx.Header("name1", "value1")
	tx.Header("name2", "value2")

	require.Equal(t, "value1", tx.header["name1"])
	require.Equal(t, "value2", tx.header["name2"])
}

func TestAccept(t *testing.T) {

	tx := Get("http://example.com")

	tx.Accept("text/plain")
	require.Equal(t, "text/plain", tx.header["Accept"])

	tx.Accept()
	require.Equal(t, "*/*", tx.header["Accept"])

	tx.Accept("application/json", "application/xml")
	require.Equal(t, "application/json;q=1.0, application/xml;q=0.9", tx.header["Accept"])

	tx.Accept("application/json", "application/xml", "text/plain")
	require.Equal(t, "application/json;q=1.0, application/xml;q=0.9, text/plain;q=0.8", tx.header["Accept"])
}

func TestContentType(t *testing.T) {

	tx := Get("http://example.com")

	tx.ContentType("text/plain")
	require.Equal(t, "text/plain", tx.header["Content-Type"])

	tx.ContentType("application/json")
	require.Equal(t, "application/json", tx.header["Content-Type"])

	tx.ContentType("tex/html")
	require.Equal(t, "tex/html", tx.header["Content-Type"])
}

func TestQuery(t *testing.T) {

	tx := Get("http://example.com")

	tx.Query("name1", "value1")
	tx.Query("name2", "value2")

	require.Equal(t, "value1", tx.query.Get("name1"))
	require.Equal(t, "value2", tx.query.Get("name2"))
}

func TestQuery_Multiple(t *testing.T) {

	tx := Get("http://example.com")

	tx.Query("name", "value1")
	tx.Query("name", "value2")
	tx.Query("name", "value3")

	require.Equal(t, []string{"value1", "value2", "value3"}, tx.query["name"])
}

func TestForm(t *testing.T) {

	tx := Get("http://example.com")

	tx.Form("name1", "value1")
	tx.Form("name2", "value2")

	require.Equal(t, "value1", tx.form.Get("name1"))
	require.Equal(t, "value2", tx.form.Get("name2"))
}

func TestBody_Text(t *testing.T) {

	tx := Get("http://example.com")

	tx.Body("Test Value")
	require.Equal(t, "Test Value", tx.body)
	require.Equal(t, "text/plain", tx.header["Content-Type"])

}

func TestBody_JSON(t *testing.T) {

	tx := Get("http://example.com")

	complex1 := []int{1, 2, 3, 4}

	tx.JSON(complex1)
	require.Equal(t, complex1, tx.body)
	require.Equal(t, "application/json", tx.header["Content-Type"])

}

func TestBody_XML(t *testing.T) {

	tx := Get("http://example.com")

	complex2 := []int{5, 6, 7, 8}

	tx.XML(complex2)
	require.Equal(t, complex2, tx.body)
	require.Equal(t, "application/xml", tx.header["Content-Type"])
}

func TestTxn(t *testing.T) {
	var txn Transaction
	require.NotNil(t, txn)
}

func TestBearCaps(t *testing.T) {

	tx := Get("bear:?t=123456789101112&u=http://test.com")
	err := tx.assembleBearCap()

	require.Nil(t, err)
	require.Equal(t, "Bearer 123456789101112", tx.header["Authorization"])
	require.Equal(t, "http://test.com", tx.url)
}

func TestMarshaller(t *testing.T) {

	tx1 := Get("http://example.com").
		UserAgent("Testy McTesterson").
		Accept("text/plain").
		Form("name2", "value2")

	original := tx1.MarshalMap()

	tx2 := New()
	err := tx2.UnmarshalMap(original)
	require.Nil(t, err)

	require.Equal(t, tx1.method, tx2.method)
	require.Equal(t, tx1.url, tx2.url)
	require.Equal(t, tx1.header, tx2.header)

	require.Equal(t, "text/plain", tx2.header["Accept"])
	require.Equal(t, "Testy McTesterson", tx2.header["User-Agent"])
	require.Equal(t, "application/x-www-form-urlencoded", tx2.header["Content-Type"])
}
