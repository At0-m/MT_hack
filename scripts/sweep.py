#!/usr/bin/env python3
"""Stepwise load/stress sweep. A passing tested point is NOT a proven maximum."""
import argparse
import json
import subprocess
import sys
import time
from pathlib import Path
from common import ROOT
p=argparse.ArgumentParser()
p.add_argument('--rates',default='50,100,200,300,400')
p.add_argument('--script',default='forecast-day')
p.add_argument('--duration',default='5m')
p.add_argument('--warmup',default='2m')
p.add_argument('--replicas',choices=('1','2'),default='1')
a=p.parse_args()
rates=[int(x) for x in a.rates.split(',')]
if not rates or any(r<=0 or r>10000 for r in rates) or rates!=sorted(set(rates)):p.error('Rates must be ascending unique integers, 1..10000')
root=ROOT/'benchmarks'/'results'/time.strftime('sweep-%Y%m%dT%H%M%SZ',time.gmtime());root.mkdir(parents=True,exist_ok=False)
rows=[]
for i,rate in enumerate(rates):
    out=root/f'{rate}rps'
    command=[sys.executable,str(ROOT/'scripts'/'benchmark.py'),'--script',a.script,'--rate',str(rate),'--duration',a.duration,'--warmup',a.warmup,'--replicas',a.replicas,'--output',str(out)]
    if i:command+=['--no-start']
    code=subprocess.run(command,cwd=ROOT).returncode
    report=json.loads((out/'report.json').read_text()) if (out/'report.json').exists() else {}
    rows.append({'rate':rate,'exit_code':code,'http_slo_passed':report.get('http_slo_passed',False),'report':str(out.relative_to(ROOT))})
    if code:break
(root/'sweep.json').write_text(json.dumps(rows,indent=2)+'\n')
passed=[r['rate'] for r in rows if r['http_slo_passed']]
(root/'sweep.md').write_text('# Stepwise capacity evidence\n\nHighest tested HTTP-SLO-passing rate: '+str(max(passed) if passed else 'none')+' RPS.\n\nNot an exact capacity maximum. Confirm CPU/RAM/DB/generator headroom, then run a 30-60 minute soak at 60-70% of the accepted sustained rate. If every step passed, saturation was not reached.\n')
print(root)
sys.exit(0 if rows and all(r['exit_code']==0 for r in rows) else 1)
