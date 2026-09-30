#!/usr/bin/env python3
"""Validate real Go-generated API fixtures against the bundled OpenAPI schemas.
Requires: python3 -m pip install pyyaml jsonschema
This is a schema/contract test, not a substitute for real HTTP/ONNX/PostGIS tests.
"""
from pathlib import Path
import json
import sys
import yaml
from jsonschema import Draft7Validator, FormatChecker
ROOT = Path(__file__).resolve().parents[1]
def converted(value):
    if isinstance(value, list): return [converted(v) for v in value]
    if not isinstance(value, dict): return value
    result = {k: converted(v) for k,v in value.items() if k not in {'nullable','discriminator','example','xml'}}
    for key in ['exclusiveMinimum','exclusiveMaximum']:
        if isinstance(result.get(key), bool):
            inclusive = 'minimum' if key == 'exclusiveMinimum' else 'maximum'
            active = result.pop(key)
            if active and inclusive in result: result[key] = result.pop(inclusive)
    if value.get('nullable'): return {'anyOf': [result, {'type':'null'}]}
    return result

def main():
    main = (ROOT/'openapi/openapi.yaml').read_bytes()
    for name in ['backend/openapi/openapi.yaml','frontend/openapi/openapi.yaml']:
        assert (ROOT/name).read_bytes() == main, f'OpenAPI copy drift: {name}'
    spec = yaml.safe_load(main)
    operations=[]; references=0
    def walk(value):
        nonlocal references
        if isinstance(value, list):
            for x in value: walk(x)
        elif isinstance(value,dict):
            if '$ref' in value:
                ref=value['$ref']; assert ref.startswith('#/'), ref
                node=spec
                for key in ref[2:].split('/'): node=node[key.replace('~1','/').replace('~0','~')]
                references+=1
            for v in value.values(): walk(v)
    walk(spec)
    for path,item in spec['paths'].items():
        for method, operation in item.items():
            if method.lower() in {'get','post','put','delete','patch','head','options'}:
                operations.append(operation['operationId'])
    assert len(operations)==len(set(operations)), 'Duplicate operationId'
    base=converted(spec)
    counts=0
    examples=ROOT/'integration/fixtures'
    for file in sorted(examples.glob('*.json')):
        name=file.stem
        if name=='snapshot': schema='Snapshot'
        elif name=='geometry': schema='RouteGeometry'
        elif name.endswith('-summary'): schema='SummaryResponse'
        elif name.endswith('-request'): schema='CalculationDescriptor'
        else: schema='CalculationResponse'
        validator=Draft7Validator({'$ref':'#/components/schemas/'+schema,'components':base['components']},format_checker=FormatChecker())
        errors=sorted(validator.iter_errors(json.loads(file.read_text())),key=lambda e:str(list(e.path)))
        if errors:
            print(file.name, schema)
            for error in errors[:5]: print(list(error.path),error.message[:500])
            raise ValueError('OpenAPI fixture mismatch')
        counts+=1
    print(f'PASS: {counts} generated fixtures; {len(operations)} operations; {references} local references; 3 identical OpenAPI copies.')
if __name__=='__main__': main()
