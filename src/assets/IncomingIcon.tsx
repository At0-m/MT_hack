export function IncomingIcon({name,className=''}:{name:string;className?:string}){
 return <span className={`incoming-icon ${className}`} aria-hidden="true"><img src={`/assets/icons/incoming/${name}`} alt=""/></span>;
}
