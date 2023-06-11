/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
)

var plog = logrus.WithField("component", "PolicyEngine")
var MyPolicyHandler PolicyHandler

type MbgState struct {
	mbgPeers *[]string
}

type PolicyHandler struct {
	SubscriptionMap map[string][]string
	accessControl   *AccessControl
	loadBalancer    *LoadBalancer
	mbgState        MbgState
}

func (pH PolicyHandler) Routes(r *chi.Mux) chi.Router {

	r.Get("/", pH.policyWelcome)

	r.Route("/"+event.NewConnectionRequest, func(r chi.Router) {
		r.Post("/", pH.newConnectionRequest) // New connection request
	})

	r.Route("/"+event.AddPeerRequest, func(r chi.Router) {
		r.Post("/", pH.addPeerRequest) // New peer request
	})

	r.Route("/"+event.RemovePeerRequest, func(r chi.Router) {
		r.Post("/", pH.removePeerRequest) // Remove peer request
	})

	r.Route("/"+event.NewRemoteService, func(r chi.Router) {
		r.Post("/", pH.newRemoteService) // New remote service request
	})
	r.Route("/"+event.RemoveRemoteService, func(r chi.Router) {
		r.Post("/", pH.removeRemoteServiceRequest) // Remove remote service request
	})
	r.Route("/"+event.ExposeRequest, func(r chi.Router) {
		r.Post("/", pH.exposeRequest) // New expose request
	})

	r.Route("/acl", func(r chi.Router) {
		r.Get("/", pH.accessControl.GetRuleReq)
		r.Post("/add", pH.accessControl.AddRuleReq) // Add ACL Rule
		r.Post("/delete", pH.accessControl.DelRuleReq)
	})

	r.Route("/lb", func(r chi.Router) {
		r.Get("/", pH.loadBalancer.GetPolicyReq)
		r.Post("/add", pH.loadBalancer.SetPolicyReq)       // Add LB Policy
		r.Post("/delete", pH.loadBalancer.DeletePolicyReq) // Delete LB Policy

	})
	return r
}

func exists(slice []string, entry string) (int, bool) {
	for i, e := range slice {
		if e == entry {
			return i, true
		}
	}
	return -1, false
}

func (pH PolicyHandler) addPeer(peerMbg string) {
	_, exist := exists(*pH.mbgState.mbgPeers, peerMbg)
	if exist {
		return
	}
	*pH.mbgState.mbgPeers = append(*pH.mbgState.mbgPeers, peerMbg)
	plog.Infof("Added Peer %+v", pH.mbgState.mbgPeers)
}

func (pH PolicyHandler) removePeer(peerMbg string) {
	index, exist := exists(*pH.mbgState.mbgPeers, peerMbg)
	if !exist {
		return
	}
	*pH.mbgState.mbgPeers = append((*pH.mbgState.mbgPeers)[:index], (*pH.mbgState.mbgPeers)[index+1:]...)
	plog.Infof("Removed Peer(%s, %d) %+v", peerMbg, index, *pH.mbgState.mbgPeers)
}

func (pH PolicyHandler) newConnectionRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ConnectionRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New connection request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.NewConnectionRequest])

	var action event.Action
	var targetMbg string
	var bitrate int
	for _, agent := range pH.SubscriptionMap[event.NewConnectionRequest] {
		plog.Infof("Applying Policy %s", agent)
		switch agent {
		case "AccessControl":
			if requestAttr.Direction == event.Incoming {
				action, bitrate = pH.accessControl.Lookup(requestAttr.SrcService, requestAttr.DstService, requestAttr.OtherMbg, event.Allow)
			}
		case "LoadBalancer":
			plog.Infof("Looking up loadbalancer direction %v", requestAttr.Direction)
			if requestAttr.Direction == event.Outgoing {
				// Get a list of MBGs for the service
				mbgList, err := pH.loadBalancer.GetTargetMbgs(requestAttr.DstService)
				if err != nil {
					action = event.Deny
					break
				} else {
					action = event.Allow
				}
				// Truncate mbgs from mbgList based on the policy
				var mbgValidList []string
				for _, mbg := range mbgList {
					act, _ := pH.accessControl.Lookup(requestAttr.SrcService, requestAttr.DstService, mbg, pH.accessControl.DefaultRule) //For new outgoing connections, the default is set up in the init state
					if act != event.Deny {
						mbgValidList = append(mbgValidList, mbg)
					}
				}
				// Perform loadbancing using the truncated mbgList
				targetMbg, err = pH.loadBalancer.LookupWith(requestAttr.SrcService, requestAttr.DstService, mbgValidList)
				if err != nil {
					action = event.Deny
				}
			}
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}

	plog.Infof("Response : %+v", event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH PolicyHandler) addPeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Add Peer reqest : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.AddPeerRequest])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	var action event.Action

	for _, agent := range pH.SubscriptionMap[event.AddPeerRequest] {
		switch agent {
		case "AccessControl":
			_, action, _ = pH.accessControl.RulesLookup(event.Wildcard, event.Wildcard, requestAttr.PeerMbg)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.AddPeerResp{Action: action}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

	// Update States
	if action != event.Deny {
		pH.addPeer(requestAttr.PeerMbg)
	}

}

func (pH PolicyHandler) removePeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Remove Peer request : %+v ", requestAttr)
	pH.removePeer(requestAttr.PeerMbg)
	pH.loadBalancer.RemoveMbgFromServiceMap(requestAttr.PeerMbg)
	w.WriteHeader(http.StatusOK)

}

func (pH PolicyHandler) newRemoteService(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.NewRemoteServiceAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New Remote Service request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.NewRemoteService])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	var action event.Action

	for _, agent := range pH.SubscriptionMap[event.NewRemoteService] {
		switch agent {
		case "AccessControl":
			action, _ = pH.accessControl.Lookup(event.Wildcard, requestAttr.Service, requestAttr.Mbg, event.Allow)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.NewRemoteServiceResp{Action: action}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
	}

	// Update States
	if action != event.Deny {
		pH.loadBalancer.AddToServiceMap(requestAttr.Service, requestAttr.Mbg)
	}
}

func (pH PolicyHandler) removeRemoteServiceRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.RemoveRemoteServiceAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Remove remote service request : %+v ", requestAttr)
	pH.loadBalancer.RemoveDestService(requestAttr.Service, requestAttr.Mbg)
	w.WriteHeader(http.StatusOK)

}

func (pH PolicyHandler) exposeRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ExposeRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New Expose request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.ExposeRequest])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	action := event.AllowAll
	var mbgPeers []string

	for _, agent := range pH.SubscriptionMap[event.ExposeRequest] {
		switch agent {
		case "AccessControl":
			plog.Infof("Checking accesses for %+v", pH.mbgState.mbgPeers)
			action, mbgPeers = pH.accessControl.LookupTarget(requestAttr.Service, pH.mbgState.mbgPeers)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.ExposeRequestResp{Action: action, TargetMbgs: mbgPeers}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

}

func (pH PolicyHandler) policyWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Policy Engine"))
	if err != nil {
		log.Println(err)
	}
}
func (pH PolicyHandler) init(router *chi.Mux, defaultRule event.Action) {
	pH.SubscriptionMap = make(map[string][]string)
	pH.mbgState.mbgPeers = &([]string{})
	policyList1 := []string{"AccessControl", "LoadBalancer"}
	policyList2 := []string{"AccessControl"}

	pH.accessControl = &AccessControl{DefaultRule: defaultRule}
	pH.loadBalancer = &LoadBalancer{}
	pH.accessControl.Init()
	pH.loadBalancer.Init()

	pH.SubscriptionMap[event.NewConnectionRequest] = policyList1
	pH.SubscriptionMap[event.AddPeerRequest] = policyList2
	pH.SubscriptionMap[event.NewRemoteService] = policyList2
	pH.SubscriptionMap[event.ExposeRequest] = policyList2

	plog.Infof("Subscription Map - %+v", pH.SubscriptionMap)

	routes := pH.Routes(router)

	router.Mount("/policy", routes)

}

func StartPolicyDispatcher(router *chi.Mux, defaultRule event.Action) {
	plog.Infof("Policy Engine started")
	MyPolicyHandler.init(router, defaultRule)

}
