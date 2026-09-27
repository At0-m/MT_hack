#!/usr/bin/env python3
"""Restart one API replica. Never flush kernel page caches or restart PostgreSQL."""
import json
import time
from common import API,ROOT,Compose,environment,scrape,host_metadata

def main():
    env=environment();dc=Compose(env);ids=dc.ids('api')
    if len(ids)!=1:raise RuntimeError('Cold-start measurement requires exactly one API replica')
    out=ROOT/'benchmarks'/'results'/time.strftime('cold-%Y%m%dT%H%M%SZ',time.gmtime());out.mkdir(parents=True,exist_ok=False)
    meta=host_metadata(env)
    start=time.perf_counter();dc.run('restart','api',timeout=60)
    api=API(f"http://127.0.0.1:{env.get('HTTP_PORT','8080')}")
    timings={}
    for path,key in [('/health/live','restart_command_to_live_ms'),('/health/ready','restart_command_to_ready_ms')]:
        deadline=time.monotonic()+120
        while True:
            try:
                code,_,_=api.request(path,timeout=3)
                if code==200:timings[key]=(time.perf_counter()-start)*1000;break
            except OSError:pass
            if time.monotonic()>deadline:raise RuntimeError(f'{path} timeout')
            time.sleep(.2)
    api.login(env);code,b,_=api.request('/api/v1/bootstrap')
    if code!=200:raise RuntimeError('Bootstrap failed')
    query={'selection':b['default_selection']}
    for key in ['first_forecast_after_readiness_ms','repeated_forecast_ms']:
        code,response,elapsed=api.request('/api/v1/forecasts/query',query)
        if code!=200 or len(response.get('frames',[]))!=24:raise RuntimeError(f'{key} failed: {code}')
        timings[key]=elapsed
    api.logout()
    meta.update(provenance=b['active_snapshot']['provenance'],timings=timings,
        interpretation='Readiness loads/verifies model. First forecast is prediction-cache-cold but model-session-warm. PostgreSQL/OS caches remain warm. Restart timings include Docker command overhead. Two requests are not a latency percentile distribution.')
    (out/'cold-start.json').write_text(json.dumps(meta,indent=2)+'\n')
    (out/'metrics-after.prom').write_text(scrape(ids[0],env))
    print(json.dumps(timings,indent=2));print(out)
if __name__=='__main__':main()
