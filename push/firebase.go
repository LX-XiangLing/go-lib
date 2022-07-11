package push

import (
	"context"
	"errors"
	"log"
	"math"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type FirebaseAppPush struct {
	Ctx context.Context
	App *firebase.App
}

const firebasePushPrivateKey = ``

func NewDefaultFirebaseAppPush() (*FirebaseAppPush, error) {
	ctx := context.Background()
	opts := option.WithCredentialsJSON([]byte(firebasePushPrivateKey))
	app, err := firebase.NewApp(ctx, nil, opts)
	if err != nil {
		return nil, err
	}
	return &FirebaseAppPush{
		Ctx: ctx,
		App: app,
	}, nil
}

// sendMulticastAndHandlerError sends a multicast message to all the provided tokens and catch failed tokens.
// A single sendMulticastAndHandlerError may contain up to 500 registration tokens
// return failed tokens and error
// if err != nil it can be considered that all tokens failed
func (a *FirebaseAppPush) sendMulticastAndHandlerError(messages *MulticastMessage) ([]string, error) {
	if messages == nil {
		return nil, errors.New("messages is nil")
	}
	br, err := a.sendMulticast(messages)
	if err != nil {
		return messages.Tokens, err
	}
	if br.FailureCount > 0 {
		var failedTokens []string
		for idx, resp := range br.Responses {
			if !resp.Success {
				failedTokens = append(failedTokens, messages.Tokens[idx])
			}
		}
		log.Println("Failed to send message to ", br.FailureCount, " tokens:", failedTokens)
		return failedTokens, nil
	} else {
		log.Println("Successfully sent message to all tokens:", br.SuccessCount)
	}
	return []string{}, nil
}

// divideTokenSendMulticast divide tokens into batches and send them to firebase.
// return failed tokens and error
func (a *FirebaseAppPush) divideTokenSendMulticastAndHandlerError(messages *MulticastMessage) ([]string, error) {
	var failedTokens []string
	if messages == nil {
		return nil, errors.New("messages is nil")
	}
	if messages.Tokens == nil || len(messages.Tokens) == 0 {
		return nil, errors.New("tokens is empty")
	}
	for idx := 0; idx < len(messages.Tokens); idx += 500 {
		submitTokens := messages.Tokens[idx:int64(math.Min(float64(idx+500), float64(len(messages.Tokens))))]
		if len(submitTokens) == 0 {
			continue
		}
		submitMessages := &MulticastMessage{
			Tokens:       submitTokens,
			Data:         messages.Data,
			Notification: messages.Notification,
			Android:      messages.Android,
			Webpush:      messages.Webpush,
			APNS:         messages.APNS,
		}
		onceFailedTokens, err := a.sendMulticastAndHandlerError(submitMessages)
		if err != nil {
			failedTokens = append(failedTokens, submitTokens...)
			continue
		}
		failedTokens = append(failedTokens, onceFailedTokens...)
	}
	return failedTokens, nil
}

// sendMulticast sends a multicast message to all the provided tokens.
// A single sendMulticast may contain up to 500 registration tokens
func (a *FirebaseAppPush) sendMulticast(messages *MulticastMessage) (*BatchResponse, error) {
	ctx := a.Ctx
	client, err := a.App.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	br, err := client.SendMulticast(ctx, messages)
	if err != nil {
		return nil, err
	}
	return br, nil
}
