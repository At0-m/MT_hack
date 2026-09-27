import fs from 'node:fs';
import openapiTS, { astToString } from 'openapi-typescript';
import { parse } from 'yaml';
const schema=parse(fs.readFileSync('openapi/openapi.yaml','utf8'));
// Expand property-free `required` branches for the generator, preserving the
// original contract's anyOf semantics and never modifying openapi.yaml.
const overrides=schema.components.schemas.ScenarioOverrides;
for(const branch of overrides.anyOf){branch.type='object';branch.properties=structuredClone(overrides.properties);branch.required=[...overrides.required,...branch.required];branch.additionalProperties=false;}
fs.writeFileSync('src/api/schema.d.ts',astToString(await openapiTS(schema)));
