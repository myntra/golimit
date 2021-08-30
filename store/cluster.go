package store

import (
	"github.com/myntra/golimit/gen-go/com"
	log "github.com/sirupsen/logrus"
	"github.com/uber/ringpop-go"
	"github.com/uber/ringpop-go/discovery/statichosts"
	"github.com/uber/ringpop-go/events"
	"github.com/uber/ringpop-go/forward"
	"github.com/uber/ringpop-go/swim"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/thrift"
	"math/rand"
	"strings"
	"time"
)

type ClusterInfo struct {
	Whoami  string
	Ready   bool
	Uptime  time.Duration
	Members []string
}

func (s *Store) initRingPop() {

	var err error
	s.tchannel = s.opts.tchannel

	if s.nodeId == "" {
		s.nodeId = HOSTNAME + ":" + s.opts.tchannelport
	}

	log.Info("Cluster: Initializing RingPop")
	if s.tchannel == nil {
		s.tchannel, err = tchannel.NewChannel(s.opts.clusterName, nil)
		if err != nil {
			log.Error(err)
			panic("Cluster: channel did not create successfully")
		}

	} else {
		if s.tchannel.ServiceName() != s.opts.clusterName {
			panic("Invalid Config: Cluster name should match with provided TChannel Service Name")
		}
	}
	s.ringpop = s.opts.ringpop

	if s.ringpop == nil {
		s.ringpop, err = ringpop.New(s.opts.clusterName,
			ringpop.Channel(s.tchannel),
			ringpop.Identity(s.nodeId),
			ringpop.Address(HOSTNAME+":"+s.opts.tchannelport))
		if err != nil {
			log.Fatalf("Cluster: Unable to create Ringpop: %v", err)
		}
		if err := s.tchannel.ListenAndServe(s.opts.hostAddr + ":" + s.opts.tchannelport); err != nil {
			log.Fatalf("Cluster: could not listen on given hostport: %v", err)
		}

		s.ringpop.AddListener(s)

		opts := new(swim.BootstrapOptions)

		seeds := strings.Split(s.opts.seed, ",")
		if len(seeds) > 1 {
			opts.JoinSize = 2
		} else {
			opts.JoinSize = 1
		}

		opts.JoinTimeout = 10 * time.Second
		opts.MaxJoinDuration = 60 * time.Second
		opts.DiscoverProvider = statichosts.New(seeds...)

		if _, err := s.ringpop.Bootstrap(opts); err != nil {
			log.Fatalf("Cluster: ringpop bootstrap failed: %v", err)
		}

	} else {
		nodeId, _ := s.ringpop.WhoAmI()
		if nodeId != s.nodeId {
			panic("Invalid Config: provided Ringpop Nodeid should match with " + nodeId)
		}
	}
	log.Info("Cluster: Initializing RingPop - Done")
}

func (cs *Store) registerClusterNode() error {

	log.Infof("Cluster: Registering Thrift Handler,Started")

	server := thrift.NewServer(cs.tchannel)
	server.Register(com.NewTChanStoreNodeServer(cs))

	log.Infof("Cluster: Registering Thrift Handler, Done")

	return nil
}

func (cs *Store) sendSyncCommandToNodes(syncs []*com.SyncCommand) error {

	log.Debugf("Cluster: Syncying to all nodes %+v", syncs)
	nodes, err := cs.ringpop.GetReachableMembers()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node != cs.nodeId {
			go cs.SendSyncToNode(node, syncs)
		}

	}

	return nil
}

func (cs *Store) sendSyncRateConfigToNodes(key string, rateConfig *RateConfig) error {

	log.Debugf("Cluster: Syncying Rateconfig to all nodes %+v key %s", rateConfig, key)
	nodes, err := cs.ringpop.GetReachableMembers()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node != cs.nodeId {
			go cs.SendSyncRateConfigToNode(node, key, rateConfig)
		}

	}

	return nil
}

func (cs *Store) SendSyncToNode(node string, syncs []*com.SyncCommand) error {

	log.Debugf("Cluster: Sending Incr: %+v, To Node: %s", syncs, cs.nodeId)

	var syncResult com.StoreNodeSyncKeysResult

	syncArgs := &com.StoreNodeSyncKeysArgs{
		Syncs: syncs,
	}
	req, err := ringpop.SerializeThrift(syncArgs)
	if err != nil {
		return err
	}
	forwardOptions := &forward.Options{}

	res, err := cs.ringpop.Forward(node, []string{}, req, cs.opts.clusterName, "StoreNode::SyncKeys", tchannel.Thrift, forwardOptions)
	if err == nil {
		if err = ringpop.DeserializeThrift(res, &syncResult); err != nil {

			log.Errorf("Cluster: Node: %s Sending Sync Failed error:%+v", cs.nodeId, err)
			return err
		}
	} else {
		log.Errorf("Cluster: Node: %s Sending Sync Failed , error:%+v", node, err)
	}

	log.Debugf("Cluster: %s Sending Incr Completed ", cs.nodeId)

	return nil

}

func (cs *Store) SendSyncRateConfigToNode(node string, key string, rateConfig *RateConfig) error {

	log.Debugf("Cluster: Sending RateConfig: %+v, To Node: %s", rateConfig, cs.nodeId)

	var syncResult com.StoreNodeSyncRateConfigResult

	syncArgs := &com.StoreNodeSyncRateConfigArgs{
		Key:          key,
		Window:       rateConfig.Window,
		Threshold:    rateConfig.Limit,
		Peakaveraged: rateConfig.PeakAveraged,
	}
	req, err := ringpop.SerializeThrift(syncArgs)
	if err != nil {
		return err
	}
	forwardOptions := &forward.Options{}

	res, err := cs.ringpop.Forward(node, []string{}, req, cs.opts.clusterName, "StoreNode::SyncRateConfig", tchannel.Thrift, forwardOptions)
	if err == nil {
		if err = ringpop.DeserializeThrift(res, &syncResult); err != nil {

			log.Errorf("Cluster: Node: %s Sending Sync RateConfig Failed error:%+v", cs.nodeId, err)
			return err
		}
	} else {
		log.Errorf("Cluster: Node: %s Sending Sync RateConfig Failed , error:%+v", node, err)
	}

	log.Debugf("Cluster: %s Sending Incr Completed ", cs.nodeId)

	return nil

}

func (s *Store) SyncKeys(ctx thrift.Context, syncs []*com.SyncCommand) error {
	log.Debugf("Cluster: %s Recieved Sync Request %+v ", s.nodeId, syncs)
	for _, _sync := range syncs {
		b := s.getKeyBucket(_sync.Key)
		b.Sync(_sync.Key, _sync.Count, _sync.Expiry)
	}
	log.Debugf("Cluster: %s Done Sync Request", s.nodeId)
	return nil
}

func (cs *Store) SyncRateConfig(ctx thrift.Context, key string, threshold int32, window int32, peakaveraged bool) error {

	log.Infof("Cluster:Done Sync rateconfig Request Key:%s, Threshold: %d , Window: %d, peakaveraged: %t", key, threshold, window, peakaveraged)
	cs.Lock()
	cs.rateConfig[key] = &RateConfig{Limit: threshold, Window: window,
		PeakAveraged: peakaveraged}
	cs.Unlock()
	cs.SaveRateConfig()
	return nil
}

func (cs *Store) IncrAction(ctx thrift.Context, key string, count int32, threshold int32, window int32, peakaveraged bool) (bool, error) {
	return cs.Incr(key, count, threshold, window, peakaveraged), nil
}

func (cs *Store) RateLimitGlobalAction(ctx thrift.Context, key string, count int32) (bool, error) {
	return cs.RateLimitGlobal(key, count), nil
}

func (cs *Store) HandleEvent(event events.Event) {
	//Handling Ring Changed Event we dont care about update event
	if w, ok := event.(events.RingChangedEvent); ok {
		log.Infof("Cluster Changed Added: %+v , Removed: %+v, Updated: %+v", w.ServersAdded, w.ServersRemoved, w.ServersUpdated)
		nodes, _ := cs.ringpop.GetReachableMembers()
		log.Infof("Cluster New Server List: %+v ", nodes)
		if len(w.ServersAdded) > 0 {
			go func(newnodes []string) {
				time.Sleep(time.Duration(rand.Intn(20*1000)) * time.Millisecond)
				for _, node := range newnodes {
					if node != cs.nodeId {
						for key, rateconfig := range cs.rateConfig {
							cs.SendSyncRateConfigToNode(node, key, rateconfig)
						}
					}

				}
			}(w.ServersAdded)
		}

	}
}

var clusterInfo *ClusterInfo

func (cs *Store) GetClusterInfo() *ClusterInfo {
	if clusterInfo == nil {
		clusterInfo = &ClusterInfo{}
	}
	clusterInfo.Members, _ = cs.ringpop.GetReachableMembers()
	clusterInfo.Ready = cs.ringpop.Ready()
	clusterInfo.Uptime, _ = cs.ringpop.Uptime()
	clusterInfo.Whoami, _ = cs.ringpop.WhoAmI()
	return clusterInfo
}
