// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

// ContentType is the string used in the HTTP header to designate a MIME type
const ContentType = "Content-Type"

// ContentTypePlain is the default plaintext MIME type
const ContentTypePlain = "text/plain"

// ContentTypeJSON is the standard MIME Type for JSON content
const ContentTypeJSON = "application/json"

// ContentTypeForm is the standard MIME Type for Form encoded content
const ContentTypeForm = "application/x-www-form-urlencoded"

// ContentTypeXML is the standard MIME Type for XML content
const ContentTypeXML = "application/xml"

// contentTypeNonStandardXMLText is a non-standard MIME Type that might be used by other systems for XML content
const contentTypeNonStandardXMLText = "text/xml"

// contentTypeNonStandardJSONText is a non-standard MIME Type that might be used by other systems for JSON content
const contentTypeNonStandardJSONText = "text/json"
