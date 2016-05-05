package main

import (
	"bytes"
	"encoding/gob"
	"github.com/hashicorp/serf/serf"
	"github.com/satori/go.uuid"
	"os"
	"strconv"
)

func initCluster() error {
	anakinCluster = newAnakinCluster()
	return anakinCluster.Start(false)
}

func newAnakinCluster() *AnakinCluster {
	return &AnakinCluster{
		ec: make(chan serf.Event),
	}
}

type AnakinCluster struct {
	sf   *serf.Serf
	ec   chan serf.Event
	Name string
	nameHash int
}

func (ac *AnakinCluster) Start(randomNodeName bool) error {

	sc := serf.DefaultConfig()
	sc.LogOutput = filter
	sc.MemberlistConfig.LogOutput = filter

	sc.EventCh = ac.ec

	if randomNodeName {
		sc.NodeName = uuid.NewV4().String()
	} else {
		h, err := os.Hostname()

		if err != nil {
			h = "unknown"
		}

		sc.NodeName = h + ":" + strconv.Itoa(config.ClusterPort)
	}

	ac.Name = sc.NodeName
	ac.nameHash = hash(sc.NodeName)

	if sc.Tags == nil {
		sc.Tags = make(map[string]string)
	}

	sc.MemberlistConfig.AdvertisePort = config.ClusterPort
	sc.MemberlistConfig.BindPort = config.ClusterPort

	go ac.handleClusterEvents()

	s, err := serf.Create(sc)

	if err != nil {
		return err
	}

	ac.sf = s

	if len(config.ClusterMembers) != 0 {
		n, err := ac.sf.Join(config.ClusterMembers, false)

		if err != nil {
			return err
		}

		if n == 1 {
			log.Println("Started anakin cluster, awaiting additional instances ...")
		} else {
			log.Println("Cluster join was succesful, number of anakin instances: ", n)
		}

	}

	return nil
}

func (ac *AnakinCluster) BroadcastAnakinEvent(e *AnakinEvent) error {

	e.Sender = ac.nameHash
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)

	err := enc.Encode(e)

	if err != nil {
		return err
	}

	err = ac.sf.UserEvent("ace", buffer.Bytes(), true)

	if err != nil {
		return err
	}

	return nil
}

func (ac *AnakinCluster) Shutdown(waitForIt bool) error {

	var err error

	err = ac.sf.Leave()

	if err != nil {
		return err
	}

	err = ac.sf.Shutdown()

	if err != nil {
		return err
	}

	if waitForIt {
		<-ac.sf.ShutdownCh()
	}

	return nil

}

func (ac *AnakinCluster) handleClusterEvents() {

loop:
	for {
		select {
		case event, ok := <-ac.ec:

			if event != nil {
				switch event.EventType() {

				case serf.EventMemberJoin,
					serf.EventMemberLeave,
					serf.EventMemberFailed,
					serf.EventMemberUpdate,
					serf.EventMemberReap:
					go ac.handleMemberEvent(event.(serf.MemberEvent))

				case serf.EventUser:
					go ac.handleUserEvent(event.(serf.UserEvent))

				case serf.EventQuery:
					go ac.handleQuery(event.(*serf.Query))

				}
			}

			if !ok {
				log.Println("ClusterEventHandler exiting...")
				break loop
			}

		}
	}

}

func (ac *AnakinCluster) handleMemberEvent(e serf.MemberEvent) {

	for _, m := range e.Members {
		if m.Name == ac.Name {
			return
		}
	}

	switch e.EventType() {
	case serf.EventMemberJoin:
		log.Println("Anakin instance(s) joined the cluster, ", e.Members)
	case serf.EventMemberLeave:
		log.Println("Anakin instance(s) has left the cluster, ", e.Members)
	case serf.EventMemberFailed:
		log.Println("Anakin instance(s) failing: ", e.Members)
	case serf.EventMemberUpdate:
		log.Println("Anakin instance(s) updated: ", e.Members)
	case serf.EventMemberReap:
		log.Println("Anakin instance(s) reaped: ", e.Members)
	}

}

func (ac *AnakinCluster) handleUserEvent(e serf.UserEvent) {

	if e.Name != "ace" {
		log.Println("Received non anakin related event, ignoring..., name: ", e.Name)
		return
	}

	dec := gob.NewDecoder(bytes.NewReader(e.Payload))

	var m AnakinEvent

	err := dec.Decode(&m)

	if err != nil {
		log.Println("Failed decoding anakin event, error: ", err)
		return
	}

	if m.Sender == ac.nameHash {
		return
	}



	registry.RemoteRegistryEvent(m)
}

func (ac *AnakinCluster) handleQuery(e *serf.Query) {
}

type EventType int

const (
	AppCreated EventType = iota
	AppDeleted
	AppUpdated
	SvcCreated
	SvcDeleted
	SvcUpdated
	EndpCreated
	EndpUpdated
	EndpDeleted
)

type AnakinEvent struct {
	Sender    int
	EventType EventType
	Payload   string
}