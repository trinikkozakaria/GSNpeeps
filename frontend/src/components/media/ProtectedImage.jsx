import { useEffect, useState } from "react";
import { protectedMediaRequest } from "../../lib/api/client";

export const useProtectedMedia = (path) => { const [url,setURL]=useState("");useEffect(()=>{if(!path){setURL("");return undefined}let active=true;let objectURL="";const controller=new AbortController();protectedMediaRequest(path,controller.signal).then((blob)=>{if(active&&blob instanceof Blob){objectURL=URL.createObjectURL(blob);setURL(objectURL)}}).catch(()=>active&&setURL(""));return()=>{active=false;controller.abort();if(objectURL)URL.revokeObjectURL(objectURL)}},[path]);return url; };

export const ProtectedImage=({path,...props})=>{const src=useProtectedMedia(path);return src?<img src={src} {...props}/>:null};
export const ProtectedDownloadLink=({path,fileName,children,className})=>{const href=useProtectedMedia(path);return <a href={href||undefined} download={fileName} aria-disabled={!href} className={className}>{children}</a>};
