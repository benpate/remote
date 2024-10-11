package remote

import (
	"github.com/benpate/derp"
	"github.com/benpate/turbine/queue"
)

// Consumer is a queue.Consumer that sends processes queued HTTP transactions
func Consumer(options ...Option) queue.Consumer {

	const location = "remotequeue.Consumer"

	return func(name string, arguments map[string]any) (bool, error) {

		// Only process remotequeue transactions
		if name != queueTransactionName {
			return false, nil
		}

		// Create a new Transaction using configured Options
		transaction := New().With(options...)

		// Unmarshal the arguments into the transaction
		if err := transaction.UnmarshalMap(arguments); err != nil {
			return true, derp.Wrap(err, location, "Error unmarshalling transaction", arguments)
		}

		// Send the transaction
		if err := transaction.Send(); err != nil {
			return true, derp.Wrap(err, location, "Error sending transaction", transaction)
		}

		// Success!
		return true, nil
	}
}
