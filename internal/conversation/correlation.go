package conversation

// SessionRef ties a conversation to a single message inside it. Used by
// the gateway DLP middleware to stamp audit entries with enough metadata
// for "show me what triggered this" reverse navigation.
type SessionRef struct {
	Tool        string `json:"tool"`
	SessionID   string `json:"sessionID"`
	MessageUUID string `json:"messageUUID,omitempty"`
}

// MetadataKey constants used in the audit Entry.Metadata map. Centralised
// here so binding / gateway / frontend stay in lockstep.
const (
	MetaTool        = "conv_tool"
	MetaSessionID   = "conv_session_id"
	MetaMessageUUID = "conv_message_uuid"
)

// FindEventContext locates the event referenced by `ref` within `events`
// and returns it together with up to `radius` surrounding events on each
// side. Returns (nil, false) when the UUID isn't present.
func FindEventContext(events []Event, messageUUID string, radius int) ([]Event, int, bool) {
	if messageUUID == "" {
		return nil, 0, false
	}
	target := -1
	for i, e := range events {
		if e.MessageUUID == messageUUID {
			target = i
			break
		}
	}
	if target < 0 {
		return nil, 0, false
	}
	start := target - radius
	if start < 0 {
		start = 0
	}
	end := target + radius + 1
	if end > len(events) {
		end = len(events)
	}
	slice := make([]Event, end-start)
	copy(slice, events[start:end])
	return slice, target - start, true
}
