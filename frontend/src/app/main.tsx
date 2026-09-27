import { createRoot } from 'react-dom/client';
import { App } from './App';
import '../styles/main.css';
import '../styles/integration.css';

const FIGMA_W = 1920;
const FIGMA_H = 1080;

function updateUiScale() {
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  const scale = Math.min(vw / FIGMA_W, vh / FIGMA_H, 1);
  document.documentElement.style.setProperty('--ui-scale', String(scale));
}

updateUiScale();
window.addEventListener('resize', updateUiScale);

createRoot(document.getElementById('root')!).render(<App/>);