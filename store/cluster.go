package store

import (
	"github.com/uber/tchannel-go/thrift"
	log "github.com/sirupsen/logrus"
	"github.com/uber/ringpop-go"
	"github.com/uber/ringpop-go/forward"
	"github.com/uber/tchannel-go"
	"github.com/uber/ringpop-go/swim"
	"github.com/uber/ringpop-go/discovery/statichosts"
	"bitbucket.org/myntra/golimitv3/gen-go/com"
	"github.com/uber/ringpop-go/events"
	"time"
	"math/rand"
	"strings"
)


type ClusterInfo struct {
	Whoami string
	Ready bool
	Uptime time.Duration
	Members []string
}

func (s* Store) initRingPop(){

	log.Info("Cluster: Initializing RingPop")
	ch, err := tchannel.NewChannel(s.config.ClusterName, nil)
	if err != nil {
		log.Error(err)
		panic("Cluster: channel did not create successfully")
	}
	s.tchannel = ch
	rp, err := ringpop.New(s.config.ClusterName,
		ringpop.Channel(ch),
		ringpop.Address(s.config.HostName+":"+ s.config.TChannelPort))


	if err != nil {
		log.Fatalf("Cluster: Unable to create Ringpop: %v", err)
	}
	s.ringpop=rp

	if err := ch.ListenAndServe(s.config.HostName+":"+ s.config.TChannelPort); err != nil {
		log.Fatalf("Cluster: could not listen on given hostport: %v", err)
	}

	s.ringpop.AddListener(s)
	s.nodeId= s.config.HostName+":"+ s.config.TChannelPort;
	opts := new(swim.BootstrapOptions)

	opts.DiscoverProvider = statichosts.New(strings.Split(s.config.Seed,",")...)

	if _, err := s.ringpop.Bootstrap(opts); err != nil {
		log.Fatalf("Cluster: ringpop bootstrap failed: %v", err)
	}
	log.Info("Cluster: Initializing RingPop - Done")
}

func (cs* Store) registerClusterNode() error {

	log.Infof("Cluster: Registering Thrift Handler,Started")

	server := thrift.NewServer(cs.tchannel)
	server.Register(com.NewTChanStoreNodeServer(cs))

	log.Infof("Cluster: Registering Thrift Handler, Done")

	return nil
}

func (cs*Store) sendSyncCommandToNodes(syncs []*com.SyncCommand)  error{

	log.Debugf("Cluster: Syncying to all nodes %+v",syncs)
	nodes,err:=cs.ringpop.GetReachableMembers()
	if err != nil {
		return err
	}
	for _,node:= range nodes{
		if(node!=cs.nodeId){
			go cs.SendSyncToNode(node,syncs)
		}

	}

	return  nil
}

func (cs*Store) sendSyncRateConfigToNodes(key string, rateConfig *RateConfig)  error{

	log.Debugf("Cluster: Syncying Rateconfig to all nodes %+v key %s",rateConfig, key)
	nodes,err:=cs.ringpop.GetReachableMembers()
	if err != nil {
		return err
	}
	for _,node:= range nodes{
		if(node!=cs.nodeId){
			go cs.SendSyncRateConfigToNode(node,key,rateConfig)
		}

	}

	return  nil
}

func (cs *Store) SendSyncToNode(node string, syncs []*com.SyncCommand) error{

	log.Debugf("Cluster: Sending Incr: %+v, To Node: %s",syncs,cs.nodeId)

	var syncResult com.StoreNodeSyncKeysResult


	syncArgs := &com.StoreNodeSyncKeysArgs{
		Syncs: syncs,
	}
	req, err := ringpop.SerializeThrift(syncArgs)
	if err != nil {
		return err
	}
	forwardOptions := &forward.Options{}

	res, err := cs.ringpop.Forward(node, []string{},req,cs.config.ClusterName, "StoreNode::SyncKeys", tchannel.Thrift, forwardOptions)
	if err == nil{
		if err = ringpop.DeserializeThrift(res, &syncResult); err != nil {

			log.Errorf("Cluster: Node: %s Sending Sync Failed error:%+v",cs.nodeId,err)
			return  err
		}
	}else{
		log.Errorf("Cluster: Node: %s Sending Sync Failed , error:%+v",node,err)
	}

	log.Debugf("Cluster: %s Sending Incr Completed ",cs.nodeId)

	return nil

}

func (cs *Store) SendSyncRateConfigToNode(node string, key string, rateConfig *RateConfig) error{

	log.Debugf("Cluster: Sending RateConfig: %+v, To Node: %s", rateConfig,cs.nodeId)

	var syncResult com.StoreNodeSyncRateConfigResult


	syncArgs := &com.StoreNodeSyncRateConfigArgs{
		Key: key,
		Window:rateConfig.Window,
		Threshold:rateConfig.Limit,
		Peakaveraged:rateConfig.PeakAveraged,
	}
	req, err := ringpop.SerializeThrift(syncArgs)
	if err != nil {
		return err
	}
	forwardOptions := &forward.Options{}

	res, err := cs.ringpop.Forward(node, []string{},req,cs.config.ClusterName, "StoreNode::SyncRateConfig", tchannel.Thrift, forwardOptions)
	if err == nil{
		if err = ringpop.DeserializeThrift(res, &syncResult); err != nil {

			log.Errorf("Cluster: Node: %s Sending Sync RateConfig Failed error:%+v",cs.nodeId,err)
			return  err
		}
	}else{
		log.Errorf("Cluster: Node: %s Sending Sync RateConfig Failed , error:%+v",node,err)
	}

	log.Debugf("Cluster: %s Sending Incr Completed ",cs.nodeId)

	return nil

}

func (s *Store) SyncKeys(ctx thrift.Context, syncs []*com.SyncCommand) error{
	log.Debugf("Cluster: %s Recieved Sync Request %+v ",s.nodeId,syncs)
	for _, _sync := range syncs {
		b:=s.getKeyBucket(_sync.Key)
		b.Sync(_sync.Key,_sync.Count,_sync.Expiry)
	}
	log.Debugf("Cluster: %s Done Sync Request",s.nodeId)
	return nil
}

func (cs *Store) SyncRateConfig(ctx thrift.Context, key string, threshold int32, window int32, peakaveraged bool) error{

	log.Infof("Cluster:Done Sync rateconfig Request Key:%s, Threshold: %d , Window: %d, peakaveraged: %t",key,threshold,window,peakaveraged)
	cs.rateConfig[key]=&RateConfig{Limit:threshold, Window:window,
		PeakAveraged:peakaveraged}
	cs.SaveRateConfig()
	return nil
}

func (cs *Store) IncrAction(ctx thrift.Context, key string, count int32, threshold int32, window int32, peakaveraged bool) (bool,error){
	return cs.Incr(key,count,threshold,window,peakaveraged),nil
}

func (cs *Store) RateLimitGlobalAction(ctx thrift.Context, key string, count int32) (bool, error){
	return cs.RateLimitGlobal(key,count),nil
}

func (cs *Store) HandleEvent(event events.Event){
	//Handling Ring Changed Event we dont care about update event
	if w, ok :=event.(events.RingChangedEvent); ok {
		log.Infof("Cluster Changed Added: %+v , Removed: %+v, Updated: %+v",w.ServersAdded,w.ServersRemoved,w.ServersUpdated)
		nodes,_:=cs.ringpop.GetReachableMembers()
		log.Infof("Cluster New Server List: %+v ",nodes)
		if(len(w.ServersAdded)>0){
			go func(newnodes []string){
				time.Sleep(time.Duration(rand.Intn(20*1000))*time.Millisecond)
				for _,node:= range newnodes{
					if(node!= cs.nodeId){
						for key,rateconfig:= range cs.rateConfig{
							cs.SendSyncRateConfigToNode(node,key,rateconfig)
						}
					}

				}
			}(w.ServersAdded)
		}


	}
}

var clusterInfo *ClusterInfo

func (cs *Store) GetClusterInfo() *ClusterInfo{
	if(clusterInfo==nil){
		clusterInfo=&ClusterInfo{}
	}
	clusterInfo.Members,_=cs.ringpop.GetReachableMembers()
	clusterInfo.Ready=cs.ringpop.Ready()
	clusterInfo.Uptime,_=cs.ringpop.Uptime()
	clusterInfo.Whoami,_=cs.ringpop.WhoAmI()
	return clusterInfo
}