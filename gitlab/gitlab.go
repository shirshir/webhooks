package gitlab

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"gopkg.in/go-playground/webhooks.v2"
)

// Webhook instance contains all methods needed to process events
type Webhook struct {
	provider   webhooks.Provider
	secret     string
	eventFuncs map[Event]webhooks.ProcessPayloadFunc
}

// Config defines the configuration to create a new GitHub Webhook instance
type Config struct {
	Secret string
}

// Event defines a GitHub hook event type
type Event string

// GitLab hook types
const (
	PushEvents          Event = "Push Hook"
	TagEvents           Event = "Tag Push Hook"
	IssuesEvents        Event = "Issue Hook"
	CommentEvents       Event = "Note Hook"
	MergerRequestEvents Event = "Merge Request Hook"
	WikiPageEvents      Event = "Wiki Page Hook"
	PipelineEvents      Event = "Pipeline Hook"
	BuildEvents         Event = "Build Hook"
)

// New creates and returns a WebHook instance denoted by the Provider type
func New(config *Config) *Webhook {
	return &Webhook{
		provider:   webhooks.GitLab,
		secret:     config.Secret,
		eventFuncs: map[Event]webhooks.ProcessPayloadFunc{},
	}
}

// Provider returns the current hooks provider ID
func (hook Webhook) Provider() webhooks.Provider {
	return hook.provider
}

// RegisterEvents registers the function to call when the specified event(s) are encountered
func (hook Webhook) RegisterEvents(fn webhooks.ProcessPayloadFunc, events ...Event) {

	for _, event := range events {
		hook.eventFuncs[event] = fn
	}
}

// ParsePayload parses and verifies the payload and fires off the mapped function, if it exists.
func (hook Webhook) ParsePayload(w http.ResponseWriter, r *http.Request) {

	event := r.Header.Get("X-Gitlab-Event")
	if len(event) == 0 {
		http.Error(w, "400 Bad Request - Missing X-Gitlab-Event Header", http.StatusBadRequest)
		return
	}

	gitLabEvent := Event(event)

	fn, ok := hook.eventFuncs[gitLabEvent]
	// if no event registered
	if !ok {
		return
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		http.Error(w, "Error reading Body", http.StatusInternalServerError)
		return
	}

	// If we have a Secret set, we should check the MAC
	if len(hook.secret) > 0 {

		signature := r.Header.Get("X-Gitlab-Token")

		if signature != hook.secret {
			http.Error(w, "403 Forbidden - Token missmatch", http.StatusForbidden)
			return
		}
	}

	// Make headers available to ProcessPayloadFunc as a webhooks type
	hd := webhooks.Header(r.Header)

	switch gitLabEvent {
	case PushEvents:
		var pe PushEventPayload
		json.Unmarshal([]byte(payload), &pe)
		hook.runProcessPayloadFunc(fn, pe, hd)

	case TagEvents:
		var te TagEventPayload
		json.Unmarshal([]byte(payload), &te)
		hook.runProcessPayloadFunc(fn, te, hd)

	case IssuesEvents:
		var ie IssueEventPayload
		json.Unmarshal([]byte(payload), &ie)
		hook.runProcessPayloadFunc(fn, ie, hd)

	case CommentEvents:
		var ce CommentEventPayload
		json.Unmarshal([]byte(payload), &ce)
		hook.runProcessPayloadFunc(fn, ce, hd)

	case MergerRequestEvents:
		var mre MergeRequestEventPayload
		json.Unmarshal([]byte(payload), &mre)
		hook.runProcessPayloadFunc(fn, mre, hd)

	case WikiPageEvents:
		var wpe WikiPageEventPayload
		json.Unmarshal([]byte(payload), &wpe)
		hook.runProcessPayloadFunc(fn, wpe, hd)

	case PipelineEvents:
		var pe PipelineEventPayload
		json.Unmarshal([]byte(payload), &pe)
		hook.runProcessPayloadFunc(fn, pe, hd)

	case BuildEvents:
		var be BuildEventPayload
		json.Unmarshal([]byte(payload), &be)
		hook.runProcessPayloadFunc(fn, be, hd)
	}
}

func (hook Webhook) runProcessPayloadFunc(fn webhooks.ProcessPayloadFunc, results interface{}, header webhooks.Header) {
	go func(fn webhooks.ProcessPayloadFunc, results interface{}, header webhooks.Header) {
		fn(results, header)
	}(fn, results, header)
}
