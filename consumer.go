package remote

import (
	"github.com/benpate/derp"
	"github.com/benpate/turbine/queue"
)

// Consumer is a queue.Consumer that sends processes queued HTTP transactions
func Consumer(options ...Option) queue.Consumer {

	const location = "remote.Consumer"

	return func(name string, arguments map[string]any) queue.Result {

		// Ignore transactions that are not "remote.Transaction.Send"
		if name != queueTransactionName {
			return queue.Ignored()
		}

		// Create a new Transaction using configured Options
		transaction := New().With(options...)

		// Unmarshal the arguments into the transaction
		if err := transaction.UnmarshalMap(arguments); err != nil {
			return queue.Failure(derp.Wrap(err, location, "Error unmarshalling transaction", arguments))
		}

		// Send the transaction
		if err := transaction.Send(); err != nil {
			return queue.Error(derp.Wrap(err, location, "Error sending transaction", transaction))
		}

		// Success!
		return queue.Success()
	}
}
