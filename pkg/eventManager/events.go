package eventManager

const (
	Incoming = iota
	Outgoing
)

const (
	Allow int = iota
	Deny
)

type ConnectionRequestAttr struct {
	SrcService string
	DstService string
	Direction  int
	OtherMbg   string //Optional: Would not be set if its an outgoing connection
}

type ConnectionRequestResp struct {
	Action    int
	TargetMbg string
	BitRate   int // Mbps
}

type NewRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type NewRemoteServiceResp struct {
	Action int
}

type ExposeRequestAttr struct {
	Service string
}

type ExposeRequestResp struct {
	TargetMbgs []string
}

type ServiceListRequestAttr struct {
	SrcMbg string
}

type ServiceListRequestResp struct {
	Services []string
}
