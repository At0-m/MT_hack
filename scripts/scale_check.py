#!/usr/bin/env python3
import json
import time
from common import ROOT,Compose,environment,execute,inspect_container,wait_ready

def main():
    env=environment();dc=Compose(env)
    dc.run('up','-d','--no-deps','--scale','api=2','--wait','--wait-timeout','180','api',timeout=240)
    dc.run('up','-d','--no-deps','--force-recreate','gateway',timeout=120)
    wait_ready(env)
    ids=dc.ids('api')
    if len(ids)!=2:raise RuntimeError('Expected two API replicas')
    a,b=[inspect_container(cid,env) for cid in ids]
    networks=set(a['networks']) & set(b['networks'])
    if len(networks)!=1:raise RuntimeError('Expected one common network')
    network=next(iter(networks))
    env.update(BASE_A='http://'+a['networks'][network]+':8080',BASE_B='http://'+b['networks'][network]+':8080')
    env.setdefault('API_USER','devops')
    result=execute(['docker','exec','-e','BASE_A','-e','BASE_B','-e','API_USER','-e','API_PASSWORD',ids[0],'/app/sessioncheck'],env=env,timeout=90)
    out=ROOT/'benchmarks'/'results'/time.strftime('scale-%Y%m%dT%H%M%SZ',time.gmtime());out.mkdir(parents=True,exist_ok=False)
    (out/'cross-replica.json').write_text(json.dumps({'checks':json.loads(result.stdout),'replicas':[a,b]},indent=2)+'\n')
    print(result.stdout.strip());print(out)
    print('Two replicas remain running; benchmark with --replicas 2 --no-start, or restore with make up REPLICAS=1')
if __name__=='__main__':main()
