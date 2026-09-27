let ts;try{ts=require('typescript');}catch{ts=require('../frontend/node_modules/typescript');}
const fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'../frontend');
let count=0, failures=[];
function walk(dir){for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
 const file=path.join(dir,entry.name);if(entry.isDirectory())walk(file);else if(/\.tsx?$/.test(file)&&!file.endsWith('.d.ts')){
  const source=ts.createSourceFile(file,fs.readFileSync(file,'utf8'),ts.ScriptTarget.Latest,true,file.endsWith('.tsx')?ts.ScriptKind.TSX:ts.ScriptKind.TS);
  failures.push(...source.parseDiagnostics);count++;
 }
}}
walk(path.join(root,'src'));walk(path.join(root,'tests'));walk(path.join(root,'scripts'));
const files=['src/state/calculation.ts','src/state/contractTime.ts','src/api/types.ts','src/api/schema.d.ts'].map(p=>path.join(root,p));
const program=ts.createProgram(files,{strict:true,skipLibCheck:true,target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ESNext,moduleResolution:ts.ModuleResolutionKind.Bundler,noEmit:true,types:[],lib:['lib.es2022.d.ts','lib.dom.d.ts']});
failures.push(...ts.getPreEmitDiagnostics(program));
console.log('Parsed TypeScript sources:',count);console.log('Strict API boundary typecheck: '+(failures.length?'FAILED':'PASS'));
for(const d of failures){const p=d.file&&d.start!==undefined?d.file.getLineAndCharacterOfPosition(d.start):undefined;console.log(d.file?.fileName,p?`${p.line+1}:${p.character+1}`:'',ts.flattenDiagnosticMessageText(d.messageText,'\n'));}
const out=path.resolve(__dirname,'.tmp/ts-boundary');fs.mkdirSync(out,{recursive:true});fs.writeFileSync(path.join(out,'package.json'),'{"type":"module"}');
for(const f of ['calculation','contractTime']){let text=ts.transpileModule(fs.readFileSync(path.join(root,'src/state/'+f+'.ts'),'utf8'),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ESNext}}).outputText;text=text.replace(/from '\.\/contractTime'/g,"from './contractTime.js'");fs.writeFileSync(path.join(out,f+'.js'),text);}
process.exitCode=failures.length?1:0;
