package eventManager

import "time"

type Direction int

const (
	Incoming Direction = iota
	Outgoing
)

func (d Direction) String() string {
	return [...]string{"Incoming", "Outgoing"}[d]
}

type Action int

const (
	Allow Action = iota
	Deny
	AllowAll
	AllowPartial
)

type ConnectionState int

const (
	Ongoing ConnectionState = iota
	Complete
	Denied
	DeniedPeer
)

func (a Action) String() string {
	return [...]string{"Allow", "Deny", "AllowAll", "AllowPartial"}[a]
}

const Wildcard = "*"

const (
	NewConnectionRequest = "NewConnectionRequest"
	ConnectionStatus     = "ConnectionStatus"
	AddPeerRequest       = "AddPeerRequest"
	NewRemoteService     = "NewRemoteService"
	ExposeRequest        = "ExposeRequest"
	RemovePeerRequest    = "RemovePeerRequest"
	RemoveRemoteService  = "RemoveRemoteService"
)

type ConnectionRequestAttr struct {
	SrcService string
	DstService string
	Direction  Direction
	OtherMbg   string //Optional: Would not be set if its an outgoing connection
}

type ConnectionRequestResp struct {
	Action    Action
	TargetMbg string
	BitRate   int // Mbps
}

type ConnectionStatusAttr struct {
	ConnectionId    string
	SrcService      string
	DstService      string
	IncomingBytes   int
	OutgoingBytes   int
	DestinationPeer string
	StartTstamp     time.Time
	LastTstamp      time.Time
	Direction       Direction
	State           ConnectionState
}

type NewRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type RemoveRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type NewRemoteServiceResp struct {
	Action Action
}

type ExposeRequestAttr struct {
	Service string
}

type ExposeRequestResp struct {
	Action     Action
	TargetMbgs []string
}

type AddPeerAttr struct {
	PeerMbg string
}

type AddPeerResp struct {
	Action Action
}

type RemovePeerAttr struct {
	PeerMbg string
}

type ServiceListRequestAttr struct {
	SrcMbg string
}

type ServiceListRequestResp struct {
	Action   Action
	Services []string
}

type ServiceRequestAttr struct {
	SrcMbg string
}

type ServiceRequestResp struct {
	Action Action
}
