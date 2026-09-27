#!/usr/bin/env python3
import argparse
import time
from common import ROOT,Compose,environment,execute
p=argparse.ArgumentParser();p.add_argument('--kind',choices=['cpu','heap','allocs','goroutine'],default='cpu');p.add_argument('--replica',type=int,default=0);a=p.parse_args()
env=environment();ids=Compose(env).ids('api')
if a.replica<0 or a.replica>=len(ids):p.error('Replica index out of range')
path='profile?seconds=30' if a.kind=='cpu' else a.kind
out=ROOT/'benchmarks'/'results'/time.strftime('pprof-%Y%m%dT%H%M%SZ',time.gmtime());out.mkdir(parents=True,exist_ok=False)
import subprocess
with (out/f'{a.kind}.prof').open('wb') as f:
    res=subprocess.run(['docker','exec',ids[a.replica],'/app/healthcheck','--url',f'http://127.0.0.1:9090/debug/pprof/{path}','--print','--timeout','60s'],env=env,stdout=f,timeout=70)
if res.returncode:raise SystemExit('pprof failed. Set ENABLE_PPROF=true and recreate API first.')
print(out)
