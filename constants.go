// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

// Accept is the string used in the HTTP header to request a response be encoded as a MIME type
const Accept = "Accept"

// ContentType is the string used in the HTTP header to designate a MIME type
const ContentType = "Content-Type"

// UserAgent is the string used in the HTTP header to identify the client making the request
const UserAgent = "User-Agent"

// ContentTypeActivityPub is the standard MIME type for ActivityPub content
const ContentTypeActivityPub = "application/activity+json"

// ContentTypePlain is the default plaintext MIME type
const ContentTypePlain = "text/plain"

// ContentTypeHTML is the standard MIME type for HTML content
const ContentTypeHTML = "text/html"

// ContentTypeJSON is the standard MIME Type for JSON content
const ContentTypeJSON = "application/json"

// ContentTypeJSONLD is the standard MIME Type for JSON-LD content
// https://en.wikipedia.org/wiki/JSON-LD
const ContentTypeJSONLD = "application/ld+json"

// ContentTypeJSONFeed is the standard MIME Type for JSON Feed content
// https://en.wikipedia.org/wiki/JSON_Feed
const ContentTypeJSONFeed = "application/feed+json"

// ContentTypeJSONResourceDescriptor is the standard MIME Type for JSON Resource Descriptor content
// which is used by WebFinger: https://datatracker.ietf.org/doc/html/rfc7033#section-10.2
const ContentTypeJSONResourceDescriptor = "application/jrd+json"

// ContentTypeForm is the standard MIME Type for Form encoded content
const ContentTypeForm = "application/x-www-form-urlencoded"

// ContentTypeXML is the standard MIME Type for XML content
const ContentTypeXML = "application/xml"

// ContentTypeAtomXML is the standard MIME Type for an Atom RSS feed
const ContentTypeAtomXML = "application/atom+xml"

// ContentTypeRSSXML is the standard MIME Type for a RSS feed
const ContentTypeRSSXML = "application/rss+xml"

// contentTypeNonStandardXMLText is a non-standard MIME Type that might be used by other systems for XML content
const contentTypeNonStandardXMLText = "text/xml"

// contentTypeNonStandardJSONText is a non-standard MIME Type that might be used by other systems for JSON content
const contentTypeNonStandardJSONText = "text/json"
