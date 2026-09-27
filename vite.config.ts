import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig(async ({ mode }) => {
 const env=loadEnv(mode,process.cwd(),'');
 const mock=env.VITE_API_MODE==='mock';
 return {plugins:[react(),...(mock?[{name:'explicit-development-mock',configureServer:async(server:import('vite').ViteDevServer)=>{const {mockMiddleware}=await import('./scripts/mock');server.middlewares.use(mockMiddleware());}}]:[])],server:{port:5173,strictPort:true,proxy:mock?undefined:{'/api':{target:env.API_PROXY_TARGET||'http://localhost:8080',changeOrigin:false}}},test:{include:['src/**/*.test.ts','tests/unit/**/*.test.ts']}};
});
