package remote

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {

	tx := Get("http://example.com")

	tx.Header("name1", "value1")
	tx.Header("name2", "value2")

	assert.Equal(t, "value1", tx.HeaderValues["name1"])
	assert.Equal(t, "value2", tx.HeaderValues["name2"])
}

func TestContentType(t *testing.T) {

	tx := Get("http://example.com")

	tx.ContentType("text/plain")
	assert.Equal(t, "text/plain", tx.HeaderValues["Content-Type"])

	tx.ContentType("application/json")
	assert.Equal(t, "application/json", tx.HeaderValues["Content-Type"])

	tx.ContentType("tex/html")
	assert.Equal(t, "tex/html", tx.HeaderValues["Content-Type"])
}

func TestQuery(t *testing.T) {

	tx := Get("http://example.com")

	tx.Query("name1", "value1")
	tx.Query("name2", "value2")

	assert.Equal(t, "value1", tx.QueryString.Get("name1"))
	assert.Equal(t, "value2", tx.QueryString.Get("name2"))
}

func TestForm(t *testing.T) {

	tx := Get("http://example.com")

	tx.Form("name1", "value1")
	tx.Form("name2", "value2")

	assert.Equal(t, "value1", tx.FormData.Get("name1"))
	assert.Equal(t, "value2", tx.FormData.Get("name2"))
}

func TestBody(t *testing.T) {

	tx := Get("http://example.com")

	complex1 := []int{1, 2, 3, 4}
	complex2 := []int{5, 6, 7, 8}

	tx.Body("Test Value")
	assert.Equal(t, "Test Value", tx.BodyObject)
	assert.Equal(t, "text/plain", tx.HeaderValues["Content-Type"])

	tx.JSON(complex1)
	assert.Equal(t, complex1, tx.BodyObject)
	assert.Equal(t, "application/json", tx.HeaderValues["Content-Type"])

	tx.XML(complex2)
	assert.Equal(t, complex2, tx.BodyObject)
	assert.Equal(t, "application/xml", tx.HeaderValues["Content-Type"])
}
