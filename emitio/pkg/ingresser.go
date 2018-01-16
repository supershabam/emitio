package pkg

import "context"

// Ingresser is able to provide a cancellable channel of messages to be processed
type Ingresser interface {
	// Ingress starts the consumption of messages.
	// It is undefined behaviour to call Ingress multiple times in parallel or to call Ingress
	// once more after a previous call to Ingress has completed via a returning Wait function.
	// The caller SHOULD consume the returned channel to completion
	// Once the channel has been read to completion, the caller SHOULD call the Wait function
	// which blocks until all underlaying background processes in the ingress have completed
	// and returns either nil or and error signifying that the Ingress completed successfully
	// or because of an unhandled error.
	Ingress(ctx context.Context) (<-chan string, Wait)
	// URI uniquely identifies an ingress and all its configuration.
	URI() string
}
