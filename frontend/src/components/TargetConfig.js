import React, { useState } from 'react';
import { Save, Globe, Server, Shield } from 'lucide-react';

const TargetConfig = () => {
  const [config, setConfig] = useState({
    ip: '185.40.76.46',
    port: 80,
    domain: 'mvp2001.ru',
    username: '',
    password: '',
    ssh_key: ''
  });

  const [saved, setSaved] = useState(false);

  const handleSave = () => {
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-white">Target Configuration</h2>
      
      <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
        <div className="flex items-center space-x-3 mb-6">
          <Globe className="text-blue-400" size={24} />
          <h3 className="text-lg font-semibold text-white">Target VM Settings</h3>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm text-slate-400 mb-2">IP Address</label>
            <input
              type="text"
              value={config.ip}
              onChange={(e) => setConfig({...config, ip: e.target.value})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white font-mono"
            />
          </div>
          
          <div>
            <label className="block text-sm text-slate-400 mb-2">Port</label>
            <input
              type="number"
              value={config.port}
              onChange={(e) => setConfig({...config, port: parseInt(e.target.value)})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white font-mono"
            />
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-2">Domain</label>
            <input
              type="text"
              value={config.domain}
              onChange={(e) => setConfig({...config, domain: e.target.value})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white font-mono"
            />
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-2">Username (optional)</label>
            <input
              type="text"
              value={config.username}
              onChange={(e) => setConfig({...config, username: e.target.value})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white"
              placeholder="root"
            />
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-2">Password (optional)</label>
            <input
              type="password"
              value={config.password}
              onChange={(e) => setConfig({...config, password: e.target.value})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white"
            />
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-2">SSH Key (optional)</label>
            <textarea
              value={config.ssh_key}
              onChange={(e) => setConfig({...config, ssh_key: e.target.value})}
              className="w-full bg-slate-700 border border-slate-600 rounded px-4 py-2 text-white font-mono text-sm"
              rows={3}
              placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
            />
          </div>
        </div>

        <div className="mt-6 flex items-center justify-between">
          <div className="flex items-center space-x-2 text-amber-400">
            <Shield size={18} />
            <span className="text-sm">Credentials stored locally only</span>
          </div>
          
          <button
            onClick={handleSave}
            className={`
              px-6 py-2 rounded-lg flex items-center space-x-2 transition-colors
              ${saved 
                ? 'bg-green-500 text-white' 
                : 'bg-blue-500 hover:bg-blue-600 text-white'
              }
            `}
          >
            <Save size={18} />
            <span>{saved ? 'Saved!' : 'Save Configuration'}</span>
          </button>
        </div>
      </div>

      <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
        <div className="flex items-center space-x-3 mb-4">
          <Server className="text-green-400" size={24} />
          <h3 className="text-lg font-semibold text-white">Connection Test</h3>
        </div>
        
        <div className="flex items-center space-x-4">
          <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white transition-colors">
            Test Ping
          </button>
          <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white transition-colors">
            Test HTTP
          </button>
          <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white transition-colors">
            Test SSH
          </button>
        </div>
      </div>
    </div>
  );
};

export default TargetConfig;
