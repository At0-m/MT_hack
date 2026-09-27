#!/usr/bin/env python3
"""Run k6 against THIS compose project; preserve failures and collect sanitized evidence."""
from __future__ import annotations
import argparse
import csv
import json
import os
import shutil
import subprocess
import sys
import threading
import time
from pathlib import Path
from common import API, ROOT, SCRIPTS, Compose, duration_seconds, environment, execute, host_metadata, inspect_container, parse_bytes, redact, scrape, wait_ready
from report import parse_prometheus, render

class Sampler(threading.Thread):
    def __init__(self, out:Path, env:dict, containers:list[dict], api_ids:list[str], generator:str):
        super().__init__(daemon=True);self.out=out;self.env=env;self.containers=containers;self.api_ids=api_ids;self.generator=generator;self.stop_event=threading.Event();self.errors=[]
    def run(self):
        lookup={c['name']:c for c in self.containers}
        with (self.out/'system.csv').open('w',newline='') as f, (self.out/'metrics.ndjson').open('w') as mf:
            columns=['timestamp_ms','container_id','name','service','cpu_percent_docker','cpu_limit_cores','cpu_quota_percent','memory_usage_bytes','memory_limit_bytes','pids']
            writer=csv.DictWriter(f,fieldnames=columns);writer.writeheader()
            while not self.stop_event.is_set():
                try:
                    if self.generator not in lookup:
                        try:lookup[self.generator]=inspect_container(self.generator,self.env)
                        except (RuntimeError,KeyError):pass
                    names=list(lookup)
                    raw=execute(['docker','stats','--no-stream','--format','{{json .}}']+names,env=self.env,check=False,timeout=15).stdout
                    for line in raw.splitlines():
                        d=json.loads(line);info=lookup.get(d['Name'])
                        if not info:continue
                        cpu=float(d['CPUPerc'].rstrip('%'));used,limit=d['MemUsage'].split('/')
                        cores=info['cpu_limit_cores']
                        writer.writerow(dict(timestamp_ms=time.time()*1000,container_id=info['id'],name=info['name'],service=info['service'],cpu_percent_docker=cpu,
                            cpu_limit_cores=cores,cpu_quota_percent=cpu/cores if cores else '',memory_usage_bytes=parse_bytes(used),memory_limit_bytes=parse_bytes(limit),pids=d.get('PIDs','')))
                    f.flush()
                    for cid in self.api_ids:
                        try:
                            text=scrape(cid,self.env)
                            mf.write(json.dumps({'timestamp_ms':time.time()*1000,'container_id':cid,'metrics':parse_prometheus(text)})+'\n');mf.flush()
                        except (RuntimeError,subprocess.TimeoutExpired) as e:self.errors.append(type(e).__name__)
                except (ValueError,RuntimeError,subprocess.TimeoutExpired) as e:self.errors.append(type(e).__name__)
                self.stop_event.wait(2)


def main() -> int:
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--script',choices=sorted(SCRIPTS),default='forecast-day')
    p.add_argument('--rate',type=int,default=int(os.getenv('RATE','50')))
    p.add_argument('--duration',default=os.getenv('DURATION','5m'))
    p.add_argument('--warmup',default=os.getenv('WARMUP','2m'))
    p.add_argument('--replicas',type=int,choices=(1,2),default=int(os.getenv('REPLICAS','1')))
    p.add_argument('--no-start',action='store_true',help='Use existing deployment; do not run seed or rebuild')
    p.add_argument('--output',type=Path)
    p.add_argument('--no-raw',action='store_true')
    a=p.parse_args()
    if not 1<=a.rate<=10000 or duration_seconds(a.duration)<=0 or duration_seconds(a.warmup)<0:p.error('Invalid rate/duration')
    if not shutil.which('docker'):p.error('Docker is required; no test was run')
    env=environment();dc=Compose(env)
    if not a.no_start:
        dc.run('up','-d','--build','--wait','--wait-timeout','300','--scale',f'api={a.replicas}',timeout=1200)
        dc.run('up','-d','--no-deps','--force-recreate','gateway',timeout=120)
    wait_ready(env)
    api_ids=dc.ids('api')
    if len(api_ids)!=a.replicas:raise RuntimeError(f'Expected {a.replicas} API replicas, found {len(api_ids)}')
    containers=[inspect_container(cid,env) for cid in dc.ids()]
    for c in containers:
        if c['service']=='api' and (c['cpu_limit_cores']<=0 or c['memory_limit_bytes']<=0 or c['memory_swap_limit_bytes']!=c['memory_limit_bytes']):
            raise RuntimeError('API must have effective CPU/RAM limits and memory-swap equal to RAM (no swap)')
    api_container=next(c for c in containers if c['service']=='api')
    gateway=next(c for c in containers if c['service']=='gateway')
    common=set(api_container['networks']) & set(gateway['networks'])
    if len(common)!=1:raise RuntimeError('Cannot determine unique API/gateway Docker network')
    network=next(iter(common))
    stamp=time.strftime('%Y%m%dT%H%M%SZ',time.gmtime())
    out=(a.output or ROOT/'benchmarks'/'results'/f'{stamp}-{a.script}-{a.rate}rps-{time.time_ns()%1000000}').resolve()
    out.mkdir(parents=True,exist_ok=False)
    api=API(f"http://127.0.0.1:{env.get('HTTP_PORT','8080')}")
    api.login(env);status,b,_=api.request('/api/v1/bootstrap');api.logout()
    if status!=200:raise RuntimeError(f'Bootstrap failed: {status}')
    (out/'bootstrap.json').write_text(json.dumps(b,indent=2)+'\n')
    manifest=host_metadata(env)
    manifest.update(provenance=b['active_snapshot']['provenance'],containers=containers,
        run={'script':a.script,'rate':a.rate,'duration':a.duration,'duration_seconds':duration_seconds(a.duration),'warmup':a.warmup,'replicas':a.replicas,
             'load_generator':'Docker on same host; see container resource samples','k6_image':env.get('K6_IMAGE','grafana/k6:1.8.1'),
             'route_count':env.get('ROUTE_COUNT','1'),'route_ids':env.get('ROUTE_IDS',''),'day_offset':env.get('DAY_OFFSET','0'),
             'prediction_cache_bytes':[c['settings'].get('FORECAST_CACHE_BYTES') for c in containers if c['service']=='api']})
    (out/'environment.json').write_text(json.dumps(manifest,indent=2)+'\n')
    for cid in api_ids:(out/f'metrics-before-{cid[:12]}.prom').write_text(scrape(cid,env))
    gen=f'tramflow-k6-{time.time_ns()}'
    env.update(BASE_URL='http://gateway:8080',RATE=str(a.rate),DURATION=a.duration,WARMUP=a.warmup,K6_NO_USAGE_REPORT='true')
    env.setdefault('API_USER','devops')
    keys=['BASE_URL','API_USER','API_PASSWORD','RATE','DURATION','WARMUP','K6_NO_USAGE_REPORT','ROUTE_COUNT','ROUTE_IDS','DAY_OFFSET','PRE_VUS','MAX_VUS',
          'P95_MS','WALL_P95_MS','FORECAST_P95_MS','EXPECTED_RUNTIME_MODE','EXPECTED_MODEL_VERSION','REQUEST_TIMEOUT']
    command=['docker','run','--name',gen,'--network',network,'--user',f'{os.getuid()}:{os.getgid()}',
             '--cpus',env.get('LOADGEN_CPUS','2'),'--memory',env.get('LOADGEN_MEMORY','2g'),'--memory-swap',env.get('LOADGEN_MEMORY','2g'),
             '--read-only','--tmpfs','/tmp:rw,size=64m','--cap-drop','ALL','--security-opt','no-new-privileges:true',
             '-v',f'{ROOT / "loadtest"}:/work:ro','-v',f'{out}:/out:rw']
    for key in keys:
        if key in env:command += ['-e',key] 
    command += [env.get('K6_IMAGE','grafana/k6:1.8.1'),'run']
    if not a.no_raw:command += ['--out','json=/out/raw.json.gz']
    command += [f'/work/{a.script}.js']
    print(f'Running {a.script}; artifacts: {out}',flush=True)
    sampler=Sampler(out,env,containers,api_ids,gen)
    code=1; process=None
    try:
        with (out/'k6.log').open('w') as log:
            process=subprocess.Popen(command,cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT,text=True)
            sampler.start()
            try:code=process.wait(timeout=duration_seconds(a.duration)+duration_seconds(a.warmup)+240)
            except (KeyboardInterrupt,subprocess.TimeoutExpired):
                execute(['docker','stop','-t','5',gen],env=env,check=False,timeout=15)
                try:process.wait(timeout=15)
                except subprocess.TimeoutExpired:process.kill();process.wait()
                code=130
    finally:
        sampler.stop_event.set()
        if sampler.is_alive():sampler.join(timeout=60)
        (out/'k6.exitcode').write_text(str(code)+'\n')
        final=[]
        for c in containers:
            try:final.append(inspect_container(c['id'],env))
            except RuntimeError:pass
        try:final.append(inspect_container(gen,env))
        except RuntimeError:pass
        (out/'final-state.json').write_text(json.dumps(final,indent=2)+'\n')
        (out/'sampling-errors.json').write_text(json.dumps(sampler.errors)+'\n')
        for cid in api_ids:
            try:(out/f'metrics-after-{cid[:12]}.prom').write_text(scrape(cid,env))
            except RuntimeError:pass
        (out/'backend.log').write_text(redact(dc.run('logs','--no-color','--tail','1000','api',check=False).stdout,env))
        if (out/'k6.log').exists():(out/'k6.log').write_text(redact((out/'k6.log').read_text(),env))
        execute(['docker','rm','-f',gen],env=env,check=False)
    render(out)
    print(f'k6 exit: {code}; report: {out / "benchmark.md"}',flush=True)
    return code

if __name__=='__main__':
    try:sys.exit(main())
    except (RuntimeError,ValueError,OSError,subprocess.TimeoutExpired) as e:print(f'Benchmark failed: {e}',file=sys.stderr);sys.exit(1)
