package remote

import (
	"encoding/json"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// MarshalJSON implements the json.Marhsaller interface,
// which writes the Transaction object to a JSON string.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.MarshalMap())
}

// UnmarshalJSON implements the json.Unmarshaler interface,
// which reads a Transaction object from a JSON string.
func (t *Transaction) UnmarshalJSON(data []byte) error {

	const location = "remote.Transaction.UnmarshalJSON"

	value := make(map[string]any)

	if err := json.Unmarshal(data, &value); err != nil {
		return derp.Wrap(err, location, "Error unmarshalling JSON", data)
	}

	if err := t.UnmarshalMap(value); err != nil {
		return derp.Wrap(err, location, "Error unmarshalling map", value)
	}

	return nil
}

// MarshalMap converts a Transaction object into a map[string]any
func (t *Transaction) MarshalMap() map[string]any {

	var body []byte

	if t.method != http.MethodGet {
		body, _ = t.RequestBody()
	}

	result := map[string]any{
		"method": t.method,
		"url":    t.url,
		"header": t.header,
		"query":  t.query,
		"date":   t.header["Date"],
		"body":   string(body),
	}

	return result
}

// UnmarshalMap populates a Transaction object from a map[string]any
func (t *Transaction) UnmarshalMap(value map[string]any) error {

	t.method = convert.String(value["method"])
	t.url = convert.String(value["url"])
	t.query = convert.URLValues(value["query"])
	t.form = convert.URLValues(value["form"])
	t.body = convert.String(value["body"])
	t.header = convert.MapOfString(value["header"])
	t.header["Date"] = convert.String(value["date"])

	return nil
}
