import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(async ({ mode }) => {
 const env=loadEnv(mode,process.cwd(),'');
 const mock=env.VITE_API_MODE==='mock';
 const target=env.API_PROXY_TARGET||'http://localhost:8080';
 const allowedOrigin=env.API_PROXY_ORIGIN||new URL(target).origin;
 return {
  plugins:[react(),...(mock?[{
   name:'explicit-development-mock',
   configureServer:async(server:import('vite').ViteDevServer)=>{
    const {mockMiddleware}=await import('./scripts/mock');server.middlewares.use(mockMiddleware());
   },
  }]:[])],
  server:{port:5173,host:'127.0.0.1',strictPort:true,proxy:mock?undefined:{'/api':{
   target,changeOrigin:false,
   configure(proxy: Parameters<NonNullable<import('vite').ProxyOptions['configure']>>[0]){
    // Local dev only. Browser remains same-origin to Vite; production gateway
    // preserves the real Origin and the backend still enforces CORS/CSRF.
    proxy.on('proxyReq',(request: import('node:http').ClientRequest,incoming: import('node:http').IncomingMessage)=>{
     if(incoming.headers.origin) request.setHeader('Origin',allowedOrigin);
    });
   },
  }}},
  test:{include:['src/**/*.test.ts','tests/unit/**/*.test.ts']},
 };
});
