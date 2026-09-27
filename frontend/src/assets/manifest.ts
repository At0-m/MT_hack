import type { Schema } from '../api/types';
export const icons={logo:'/assets/icons/incoming/mos logo.png',close:'/assets/icons/incoming/cross.png'};
export function moodAsset(i:Schema['UiIndicator']):{image:string;flip?:boolean}|undefined {
 if(i.key==='weather'){
  if(i.variant==='clear')return {image:'skc_d.svg'};
  if(i.variant==='rain')return {image:'ovc_ra.svg'};
  if(i.variant==='snow')return {image:'ovc_sn.svg'};
 }
 if(i.key==='trend'&&(i.variant==='down'||i.variant==='up'))return {image:'trend down.png',flip:i.variant==='up'};
 if(i.key==='fleet')return {image:'train_white.png'};
 if(i.key==='peak')return {image:i.tone==='critical'?'high crowd.png':i.tone==='warning'?'mid crowd.png':'low crowd.png'};
 if(i.key==='calendar')return {image:i.variant==='holiday'?'holliday.png':'calendar.png'};
 if(i.key==='event')return {image:'traphic light.png'};
}
