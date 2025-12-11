package remote

import (
	"encoding/json"
	"io"

	"github.com/benpate/derp"
)

func (t *Transaction) RequestURL() string {
	result := t.url

	if len(t.query) > 0 {
		result += "?" + t.query.Encode()
	}

	return result
}

// RequestBody returns the serialized body of the request as a slice of bytes.
func (t *Transaction) RequestBody() ([]byte, error) {

	const location = "remote.Transaction.RequestBody"

	// If we already have a reader for the Body, then just return that.
	switch typedValue := t.body.(type) {

	case io.Reader:
		return io.ReadAll(typedValue)

	case string:
		return []byte(typedValue), nil

	case []byte:
		return typedValue, nil
	}

	contentType := t.header[ContentType]

	// Otherwise, use the correct Marshaller, based on the ContentType of the request
	switch contentType {

	case "", ContentTypePlain:
		return []byte{}, nil

	case ContentTypeForm:
		return []byte(t.form.Encode()), nil

	case ContentTypeJSON,
		ContentTypeJSONLD,
		ContentTypeActivityPub,
		ContentTypeJSONResourceDescriptor,
		ContentTypeJSONFeed,
		contentTypeNonStandardJSONText:

		result, err := json.Marshal(t.body)

		if err != nil {
			err = derp.WrapHTTPError(err, t.request, t.response)
			err = derp.Wrap(err, location, "Error Marshalling JSON", t.body, derp.WithInternalError())
			return nil, err
		}

		return result, nil
	}

	// Fall through to here means that we have an unrecognized content type.  Return an error.
	var err error
	err = derp.NewHTTPError(t.request, t.response)
	err = derp.Wrap(err, location, "Unsupported Content-Type", contentType, derp.WithInternalError())
	return []byte{}, err
}
