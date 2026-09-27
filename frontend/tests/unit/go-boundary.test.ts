import { describe, it, expect } from 'vitest';
import {readFileSync} from 'node:fs';
import { assertCalculation, sameFactors } from '../../src/state/calculation';
import type {Calculation, Descriptor, Geometry} from '../../src/api/types';
const read=(name:string)=>JSON.parse(readFileSync(new URL(`../../../integration/fixtures/${name}.json`,import.meta.url),'utf8'));
describe('real Go-produced response fixtures',()=>{
 const geometry=read('geometry') as Geometry;
 for(const name of ['day','fleet','factors','zero','month','last-day'])it(name,()=>expect(()=>assertCalculation(read(name) as Calculation,read(name+'-request') as Descriptor,geometry)).not.toThrow());
 it('accepts normalized omitted neutral factors',()=>expect(sameFactors(undefined,{weather:1,event:1,season:1})).toBe(true));
});
