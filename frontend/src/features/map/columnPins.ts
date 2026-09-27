import type { CustomLayerInterface, Map as TransportMap } from 'maplibre-gl';
import type { ColumnPin } from './columns';
import { COLUMN_MIN_ZOOM, isTopDown } from './columns';

/** Project labels with the same public custom-layer matrix as the 3D geometry. */
export function columnPins(map: TransportMap, host: HTMLElement) {
 const overlay=document.createElement('div');overlay.className='column-pins';overlay.setAttribute('aria-hidden','true');host.append(overlay);
 let pins:{pin:ColumnPin;element:HTMLElement}[]=[];
 const layer:CustomLayerInterface={id:'column-pins',type:'custom',renderingMode:'3d',render(_gl,{defaultProjectionData}){
  const m=defaultProjectionData.mainMatrix,canvas=map.getCanvas();
  overlay.hidden=isTopDown(map.getPitch())||map.getZoom()<COLUMN_MIN_ZOOM;
  for(const {pin,element} of pins){
   const [lng,lat]=pin.coordinates,phi=lat*Math.PI/180;
   const x=(lng+180)/360,y=(1-Math.log(Math.tan(Math.PI/4+phi/2))/Math.PI)/2,z=pin.height/(40075016.68557849*Math.cos(phi));
   const w=m[3]*x+m[7]*y+m[11]*z+m[15];
   const px=(m[0]*x+m[4]*y+m[8]*z+m[12])/w,py=(m[1]*x+m[5]*y+m[9]*z+m[13])/w;
   element.hidden=w<=0||Math.abs(px)>1.1||Math.abs(py)>1.2;
   element.style.left=`${(px+1)*canvas.clientWidth/2}px`;element.style.top=`${(1-py)*canvas.clientHeight/2}px`;
  }
 }};
 return {layer,update(values:ColumnPin[]){overlay.replaceChildren();pins=values.map(pin=>{const element=document.createElement('span');element.className='column-pin';element.textContent=`${Math.round(Math.min(1,Math.max(0,pin.value))*100)}%`;element.style.setProperty('--pin-color',pin.color);overlay.append(element);return {pin,element};});},remove(){overlay.remove();}};
}
