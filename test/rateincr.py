__author__ = 'yogeshpandey'

import thread
import time
import random
from threading import Thread
from multiprocessing import Process, Value, Lock
import sys
from urllib3 import HTTPConnectionPool
import subprocess
import json
import statsd

nodes={"0":{"host": "127.0.0.1", "port": "8080"},
            "1":{"host":"127.0.0.1","port":"8081"},
            "2":{"host":"127.0.0.1","port":"8082"},
       }
threshold=1000
ttl=60
r =[0]* len(nodes)
threadscount = 10
keyscount=10
keys=[]
timefortest=10 #minutes
statsdenabled=True

if(statsdenabled):
    statsdClient = statsd.StatsClient('localhost', 8125)
control=[]
threads=[]

filename = "testlog"
report = "report"
f = open(filename, 'w')

pool=[]
pool.append( HTTPConnectionPool('127.0.0.1:8080', maxsize=100))
pool.append( HTTPConnectionPool('127.0.0.1:8081', maxsize=100))
pool.append(HTTPConnectionPool('127.0.0.1:8082', maxsize=100))


# Define a function for the thread
def hit( threadName,keys,idr,control,file):

    while control[idr]:
        id=0
        rand=random.randint(0,len(keys)-1)
        key="mkey%03d"%(rand)
        t=time.time()
        if statsdenabled:
            start = time.time()
        response=pool[id].request('POST',"/incr?K="+key+"&C=1&W=60&T=2000&P=1")
        if statsdenabled:
            dt = int((time.time() - start) * 1000)
            statsdClient.timing('golimit', dt)
        d = response.data
        ret=json.loads(d)
        if(ret["Block"]==True):
            file.write("%s %s -block\n"%(time.ctime(t),key))
        else:
            file.write("%s %s -allow\n"%(time.ctime(t),key))

        file.flush()





# Create two threads as follows
try:
    for i in range(0,keyscount):
        keys.append("mkey%03d"%(i))


    for i in range(0,threadscount):
        control.append(True)
        th= Thread( target=hit, args=("Thread-"+str(i),keys,i,control,f) )
        threads.append(th)
        th.start()
except:
    print sys.exc_info()

time.sleep(timefortest*60)
for i in range(0,threadscount):
    control[i]=False

for i in range(0,threadscount):
    threads[i].join()

f.close()

print "Generating reports"

reportData={}
for i in range(0,keyscount):
    key ="mkey%03d"%(i)
    reportData[key]={}
    reportData[key]["hit"]={}
    reportData[key]["exceed"]={}
#print reportData
with open(filename) as f:
    for line in f:
        t=line[:16]
        k=line[25:32]
        capped=line.find("block")>-1
        if(t not in reportData[k]["exceed"].keys()):
            reportData[k]["exceed"][t]=0
        if(t not in reportData[k]["hit"].keys()):
            reportData[k]["hit"][t]=0

        if(capped):
            reportData[k]["exceed"][t]=reportData[k]["exceed"][t]+1
        else:
            reportData[k]["hit"][t]=reportData[k]["hit"][t]+1


target = open(report, 'w')
for k, d1 in reportData.iteritems():
    for h, d2 in d1.iteritems():
        if(h=="hit"):
            for t, c in d2.iteritems():
                target.write("%s %s %s %s : exceed %s\n"%(t,k,h,c,reportData[k]["exceed"][t]))

target.close()

# cmd = " > %s"% (report)
# output,error = subprocess.Popen(cmd, shell=True, executable="/bin/bash", stdout=subprocess.PIPE, stderr=subprocess.PIPE).communicate()
# for i in range(0,keyscount):
#     key ="mkey%03d"%(i)
#     print "Generating reports %s" %key
#     cmd = " grep %s %s | grep -v capped  | awk -F'[ :]+' '{print $1\":\"$2\":\"$3\":\"$4\":\"$5}' | sort | uniq -c >> %s" %(key,filename,report)
#     output,error = subprocess.Popen(cmd, shell=True, executable="/bin/bash", stdout=subprocess.PIPE, stderr=subprocess.PIPE).communicate()
#     print "Donereports %s" %key