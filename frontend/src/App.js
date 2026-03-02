import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { 
  Activity, Cpu, HardDrive, Network, Shield, 
  Database, Terminal, Menu, X
} from 'lucide-react';
import Dashboard from './components/Dashboard';
import Modules from './components/Modules';
import Monitoring from './components/Monitoring';
import TargetConfig from './components/TargetConfig';

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [wsConnected, setWsConnected] = useState(false);
  const [metrics, setMetrics] = useState(null);

  useEffect(() => {
    const ws = new WebSocket(`ws://${window.location.host}/ws`);
    
    ws.onopen = () => {
      setWsConnected(true);
      ws.send(JSON.stringify({ action: 'subscribe', channel: 'metrics' }));
    };
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.channel === 'metrics') {
        setMetrics(data.data);
      }
    };
    
    ws.onclose = () => setWsConnected(false);
    ws.onerror = () => setWsConnected(false);
    
    return () => ws.close();
  }, []);

  const menuItems = [
    { path: '/', icon: Activity, label: 'Dashboard' },
    { path: '/modules', icon: Terminal, label: 'Модули нагрузки' },
    { path: '/monitoring', icon: Cpu, label: 'Мониторинг' },
    { path: '/target', icon: Network, label: 'Целевая ВМ' },
  ];

  return (
    <Router>
      <div className="min-h-screen bg-slate-900">
        <header className="bg-slate-800 border-b border-slate-700 sticky top-0 z-50">
          <div className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center space-x-3">
              <button 
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="lg:hidden text-slate-400 hover:text-white"
              >
                {sidebarOpen ? <X size={24} /> : <Menu size={24} />}
              </button>
              <div className="flex items-center space-x-2">
                <Activity className="text-blue-500" size={28} />
                <h1 className="text-xl font-bold text-white">
                  DevOps Load Platform
                </h1>
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2 text-sm">
                <span className="text-slate-400">WebSocket:</span>
                <span className={`px-2 py-1 rounded text-xs font-medium ${
                  wsConnected 
                    ? 'bg-green-500/20 text-green-400' 
                    : 'bg-red-500/20 text-red-400'
                }`}>
                  {wsConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
              <div className="text-sm text-slate-400">
                Target: <span className="text-blue-400">185.40.76.46</span>
              </div>
            </div>
          </div>
        </header>

        <div className="flex">
          <aside className={`
            fixed lg:static inset-y-0 left-0 z-40
            w-64 bg-slate-800 border-r border-slate-700
            transform transition-transform duration-200 ease-in-out
            ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
            pt-16 lg:pt-0
          `}>
            <nav className="p-4 space-y-1">
              {menuItems.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  onClick={() => setSidebarOpen(false)}
                  className="flex items-center space-x-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-700 hover:text-white transition-colors"
                >
                  <item.icon size={20} />
                  <span>{item.label}</span>
                </Link>
              ))}
            </nav>

            {metrics && (
              <div className="p-4 border-t border-slate-700">
                <h3 className="text-xs font-semibold text-slate-500 uppercase mb-3">
                  System Stats
                </h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-400">CPU</span>
                    <span className={`font-mono ${
                      metrics.cpu.usage_percent > 80 ? 'text-red-400' : 'text-green-400'
                    }`}>
                      {metrics.cpu.usage_percent.toFixed(1)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">RAM</span>
                    <span className={`font-mono ${
                      metrics.memory.usage_percent > 80 ? 'text-red-400' : 'text-green-400'
                    }`}>
                      {metrics.memory.usage_percent.toFixed(1)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Disk</span>
                    <span className="font-mono text-blue-400">
                      {metrics.disk.usage_percent.toFixed(1)}%
                    </span>
                  </div>
                </div>
              </div>
            )}
          </aside>

          <main className="flex-1 p-6 overflow-auto">
            <Routes>
              <Route path="/" element={<Dashboard metrics={metrics} />} />
              <Route path="/modules" element={<Modules />} />
              <Route path="/monitoring" element={<Monitoring metrics={metrics} />} />
              <Route path="/target" element={<TargetConfig />} />
            </Routes>
          </main>
        </div>
      </div>
    </Router>
  );
}

export default App;
