// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

const (

	// ContentType is the string used in the HTTP header to designate a MIME type
	ContentType = "Content-Type"

	// ContentTypePlain is the default plaintext MIME type
	ContentTypePlain = "text/plain"

	// ContentTypeJSON is the standard MIME Type for JSON content
	ContentTypeJSON = "application/json"

	// ContentTypeForm is the standard MIME Type for Form encoded content
	ContentTypeForm = "application/x-www-form-urlencoded"

	// ContentTypeXML is the standard MIME Type for XML content
	ContentTypeXML = "application/xml"
)
