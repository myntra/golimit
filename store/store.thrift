namespace go com

struct SyncCommand{
    1:string key
    2:i32   count
    3:i64   expiry
    4:bool  force
}


service StoreNode {
        void SyncKeys(1:list<SyncCommand> syncs)
        void SyncRateConfig(1:string key,2:i32 threshold,3:i32 window,4:bool peakaveraged)
        bool IncrAction(1:string key,2:i32 count,3:i32 threshold,4:i32 window,5:bool peakaveraged)
        bool RateLimitGlobalAction(1:string key,2:i32 count)
}