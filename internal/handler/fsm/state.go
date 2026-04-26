package fsm

type State string

const (
	StateIdle             State = "idle"
	StateAwaitNickname    State = "await_nickname"
	StateAwaitAge         State = "await_age"
	StateAwaitCity        State = "await_city"
	StateAwaitPhoto       State = "await_photo"
	StateAwaitDescription State = "await_description"
	StateReady            State = "ready"
	StateAwaitSnapPhoto   State = "await_snap_photo"
	StateAwaitSnapCaption State = "await_snap_caption"
)
