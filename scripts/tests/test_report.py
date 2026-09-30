import json
import sys
import tempfile
import unittest
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from common import parse_bytes,duration_seconds,redact
from report import parse_prometheus,percentile_from_histogram,render

class ReportTests(unittest.TestCase):
    def test_units_and_durations(self):
        self.assertEqual(parse_bytes('128MiB'),134217728)
        self.assertEqual(parse_bytes('2 GiB'),2147483648)
        self.assertEqual(duration_seconds('2m'),120)
        with self.assertRaises(ValueError):duration_seconds('tomorrow')
    def test_secrets_are_redacted(self):
        self.assertNotIn('long-secret-value',redact('password long-secret-value',{'API_PASSWORD':'long-secret-value'}))
    def test_histogram_delta(self):
        before=parse_prometheus('x_bucket{le="0.1"} 10\nx_bucket{le="0.2"} 20\nx_bucket{le="+Inf"} 20')
        after=parse_prometheus('x_bucket{le="0.1"} 20\nx_bucket{le="0.2"} 40\nx_bucket{le="+Inf"} 40')
        self.assertAlmostEqual(percentile_from_histogram(before,after,'x'),.19)
        self.assertIsNone(percentile_from_histogram(after,after,'x'))
        self.assertIsNone(percentile_from_histogram(after,before,'x'))
    def sample(self,directory,code=0,drops=0,failed=0):
        env={'commit':'unit-test-not-real','dirty':False,'utc':'unit-test','source_sha256':'test','run':{'script':'forecast-day','rate':10,'duration_seconds':10,'warmup':'2m'},'provenance':{'runtime_mode':'test-fixture'}}
        (directory/'environment.json').write_text(json.dumps(env))
        metrics={
            'business_requests':{'values':{'count':5000}},
            'business_requests{phase:measurement}':{'values':{'count':100},'thresholds':{'count>0':{'ok':True}}},
            'business_successes{phase:measurement}':{'values':{'count':100-failed}},
            'business_latency{phase:measurement}':{'values':{'med':8,'p(95)':10,'p(99)':12}},
            'business_client_wall{phase:measurement}':{'values':{'p(95)':11}},
            'business_failures{phase:measurement}':{'values':{'rate':failed/100},'thresholds':{'rate<0.001':{'ok':failed==0}}},
            'measurement_request_start_unix_ms':{'values':{'min':1000}},
            'measurement_request_end_unix_ms':{'values':{'max':11000}},
            'dropped_iterations{scenario:measure}':{'values':{'count':drops}},
        }
        (directory/'k6-summary.json').write_text(json.dumps({'metrics':metrics}))
        (directory/'k6.exitcode').write_text(str(code))
    def test_warmup_does_not_pollute_rps(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path);r=render(path)
            self.assertEqual(r['requests'],100);self.assertEqual(r['completed_rps_including_drain'],10)
            self.assertEqual(r['successful_rps_including_drain'],10)
            self.assertTrue(r['http_slo_passed'])
            self.assertEqual(r['resources'],{})
            self.assertIn('REVIEW_REQUIRED',r['resource_verdict'])
    def test_semantic_failure_rejects_http_slo(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path,failed=1)
            report=render(path)
            self.assertFalse(report['http_slo_passed'])
            self.assertEqual(report['successful_rps_including_drain'],9.9)
    def test_functional_report_is_not_capacity(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path)
            manifest=json.loads((path/'environment.json').read_text())
            manifest['run']['script']='smoke'
            (path/'environment.json').write_text(json.dumps(manifest))
            (path/'k6-summary.json').write_text(json.dumps({'metrics':{'checks':{'thresholds':{'rate==1':{'ok':True}}}}}))
            result=render(path)
            self.assertTrue(result['functional_passed'])
            self.assertFalse(result['http_slo_passed'])
            self.assertNotIn('target_rps',result)
    def test_cpu_is_normalized_by_quota_and_warmup_excluded(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path)
            (path/'system.csv').write_text('timestamp_ms,container_id,name,service,cpu_quota_percent,memory_usage_bytes\n0,api,api,api,999,999999999\n2000,api,api,api,70,1048576\n3000,api,api,api,80,2097152\n')
            entry=render(path)['resources']['api']
            self.assertEqual(entry['cpu_quota_percent']['avg'],75)
            self.assertEqual(entry['docker_working_set_bytes']['max'],2097152)
    def test_exit_failure_is_never_success(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path,code=99);self.assertFalse(render(path)['http_slo_passed'])
    def test_dropped_load_is_not_a_pass(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path,drops=1);self.assertFalse(render(path)['http_slo_passed'])
    def test_missing_summary_is_not_a_pass(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d);self.sample(path);(path/'k6-summary.json').unlink();r=render(path)
            self.assertFalse(r['http_slo_passed']);self.assertIsNone(r['completed_rps_including_drain'])

if __name__=='__main__':unittest.main()
