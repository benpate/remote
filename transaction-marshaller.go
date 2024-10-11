package remote

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.MarshalMap())
}

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

func (t *Transaction) MarshalMap() map[string]any {

	return map[string]any{
		"method": t.method,
		"url":    t.url,
		"header": t.header,
		"query":  t.query,
		"form":   t.form,
		"body":   convert.String(t.body),
	}
}

func (t *Transaction) UnmarshalMap(value map[string]any) error {

	t.method = convert.String(value["method"])
	t.url = convert.String(value["url"])
	t.header = convert.MapOfString(value["header"])
	t.query = convert.URLValues(value["query"])
	t.form = convert.URLValues(value["form"])
	t.body = convert.String(value["body"])

	return nil
}
