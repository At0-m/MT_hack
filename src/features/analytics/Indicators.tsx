import { useState } from 'react';
import type { CSSProperties } from 'react';
import type { Schema } from '../../api/types';
import { moodAsset } from '../../assets/manifest';
const colors={positive:'#15ff00',warning:'#d9c000',critical:'#ff2424',info:'#7ce0ff',neutral:'#bbc5d2'};
function Glyph({item}:{item:Schema['UiIndicator']}){const asset=moodAsset(item);return asset?<img className={`mood-glyph ${asset.flip?'flipped':''} ${asset.image.endsWith('.png')?'raster':''}`} src={'/assets/icons/incoming/'+asset.image} alt=""/>:<span className="icon-placeholder"/>;}
export function Indicators({items}:{items:Schema['UiIndicator'][]}) {
 const [opened,setOpened]=useState<string>();
 return <div className="moods">{items.filter(i=>!(i.variant==='none'&&i.icon==='none')).map(i=>{const key=`${i.key}:${i.variant}:${i.tone}`;return <div key={key} className={`mood tone-${i.tone}`} style={{'--mood-color':colors[i.tone]} as CSSProperties} onMouseEnter={()=>setOpened(key)} onMouseLeave={()=>setOpened(undefined)}><button aria-label={i.title} aria-describedby={opened===key?`mood-${key}`:undefined} onFocus={e=>{if(e.currentTarget.matches(':focus-visible'))setOpened(key);}} onBlur={()=>setOpened(undefined)} onKeyDown={e=>{if(e.key==='Escape')setOpened(undefined);}}><span className="mood-disc"><Glyph item={i}/></span></button>{opened===key&&<div className="mood-description" id={`mood-${key}`} role="tooltip"><strong>{i.title}</strong><p>{i.text}</p>{!moodAsset(i)&&<small>Нет иконки: {i.key}/{i.variant}</small>}</div>}</div>;})}</div>;
}
export function PlainIndicators({items}:{items:Schema['UiIndicator'][]}) {
 return <div className="plain-moods" aria-label="Активные индикаторы">{items.filter(i=>!(i.variant==='none'&&i.icon==='none')).map(i=><span key={`${i.key}:${i.variant}:${i.tone}`} title={`${i.title}: ${i.text}`} aria-label={i.title}><Glyph item={i}/></span>)}</div>;
}
