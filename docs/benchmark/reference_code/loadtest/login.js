import http from 'k6/http';
import {check,sleep} from 'k6';
import {BASE,params,decode} from './lib/runner.js';
export const options={vus:1,duration:__ENV.DURATION||'30s',thresholds:{checks:['rate==1'],http_req_failed:['rate==0']},systemTags:['method','name','status'],summaryTrendStats:['avg','med','p(95)','p(99)','max']};
export default function(){
  const res=http.post(`${BASE}/api/v1/auth/login`,JSON.stringify({username:__ENV.API_USER,password:__ENV.API_PASSWORD}),params(null,'login'));
  check(res,{'login succeeds':r=>r.status===200&&decode(r)?.authenticated===true});
  sleep(2.1);
}
export function handleSummary(data){return {'/out/k6-summary.json':JSON.stringify({format:'tramflow-login-v1',metrics:data.metrics,note:'Closed-loop low-rate auth test, not service throughput'},null,2)}}
