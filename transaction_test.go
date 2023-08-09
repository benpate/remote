package remote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeader(t *testing.T) {

	tx := Get("http://example.com")

	tx.Header("name1", "value1")
	tx.Header("name2", "value2")

	require.Equal(t, "value1", tx.HeaderValues["name1"])
	require.Equal(t, "value2", tx.HeaderValues["name2"])
}

func TestAccept(t *testing.T) {

	tx := Get("http://example.com")

	tx.Accept("text/plain")
	require.Equal(t, "text/plain", tx.HeaderValues["Accept"])

	tx.Accept()
	require.Equal(t, "*/*", tx.HeaderValues["Accept"])

	tx.Accept("application/json", "application/xml")
	require.Equal(t, "application/json;q=1.0, application/xml;q=0.9", tx.HeaderValues["Accept"])

	tx.Accept("application/json", "application/xml", "text/plain")
	require.Equal(t, "application/json;q=1.0, application/xml;q=0.9, text/plain;q=0.8", tx.HeaderValues["Accept"])
}

func TestContentType(t *testing.T) {

	tx := Get("http://example.com")

	tx.ContentType("text/plain")
	require.Equal(t, "text/plain", tx.HeaderValues["Content-Type"])

	tx.ContentType("application/json")
	require.Equal(t, "application/json", tx.HeaderValues["Content-Type"])

	tx.ContentType("tex/html")
	require.Equal(t, "tex/html", tx.HeaderValues["Content-Type"])
}

func TestQuery(t *testing.T) {

	tx := Get("http://example.com")

	tx.Query("name1", "value1")
	tx.Query("name2", "value2")

	require.Equal(t, "value1", tx.QueryString.Get("name1"))
	require.Equal(t, "value2", tx.QueryString.Get("name2"))
}

func TestForm(t *testing.T) {

	tx := Get("http://example.com")

	tx.Form("name1", "value1")
	tx.Form("name2", "value2")

	require.Equal(t, "value1", tx.FormData.Get("name1"))
	require.Equal(t, "value2", tx.FormData.Get("name2"))
}

func TestBody_Text(t *testing.T) {

	tx := Get("http://example.com")

	tx.Body("Test Value")
	require.Equal(t, "Test Value", tx.BodyObject)
	require.Equal(t, "text/plain", tx.HeaderValues["Content-Type"])

}

func TestBody_JSON(t *testing.T) {

	tx := Get("http://example.com")

	complex1 := []int{1, 2, 3, 4}

	tx.JSON(complex1)
	require.Equal(t, complex1, tx.BodyObject)
	require.Equal(t, "application/json", tx.HeaderValues["Content-Type"])

}

func TestBody_XML(t *testing.T) {

	tx := Get("http://example.com")

	complex2 := []int{5, 6, 7, 8}

	tx.XML(complex2)
	require.Equal(t, complex2, tx.BodyObject)
	require.Equal(t, "application/xml", tx.HeaderValues["Content-Type"])
}
