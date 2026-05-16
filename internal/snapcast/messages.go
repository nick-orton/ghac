package snapcast

// SnapClient holds the current state of a single SnapCast client.
type SnapClient struct {
	ID     string
	Name   string
	Volume int  // 0–100
	Muted  bool
}

// MsgClientsUpdated is emitted when one or more SnapCast clients change
// volume, mute state, or connection status. It carries the full updated
// client list so the UI can replace its state wholesale.
type MsgClientsUpdated struct {
	Clients []SnapClient
}

// MsgError is sent when the SnapCast connection is lost or a fatal error
// occurs during the notification listener.
type MsgError struct {
	Err error
}
